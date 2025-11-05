import {secret} from "./secrets";
import {allowedOrigins, apiDomain, authDomain, domain, myRouter, subdomain} from "./domain";
import {bcvBucket, receiptsBucket, webAssetsBucket} from "./storage";
import {isLocal, isrPrefix, PROD_STAGE} from "./util";
import {Output} from "@pulumi/pulumi";


const processUserFunction = new sst.aws.Function("ProcessUser", {
  link: [secret.secretTursoUrl, secret.telegramBotToken, secret.telegramBotApiKey],
  handler: "packages/backend/kyo-repo/cmd/process-user/process-user.go",
  runtime: "go",
});


export const auth = new sst.aws.Auth("AuthServer", {
  domain: {
    name: authDomain,
    // dns: sst.aws.dns({override: true}),
  },
  forceUpgrade: "v2",
  issuer: {
    handler: "packages/auth/index.handler",
    link: [
      secret.githubClientId,
      secret.githubClientSecret,
      secret.googleClientId,
      secret.googleClientSecret,
      secret.posthogApiKey,
      processUserFunction,
    ],
    environment: {
      API_URL: `https://${apiDomain}`,
    },
  },
});


const isrGenFunction = new sst.aws.Function("IsrGenFunction", {
  url: true,
  link: [webAssetsBucket, secret.secretTursoUrl],
  environment: {
    ISR_PREFIX: isrPrefix
  },
  handler: "packages/backend/kyo-repo/cmd/isr-gen/isr-gen.go",
  runtime: "go",
});


const api = new sst.aws.ApiGatewayV2("API", {
  // domain: {
  //   name: apiDomain,
  //   dns: sst.aws.dns({override: true}),
  // },
  accessLog: {
    retention: "1 month",
  },
  cors: {
    allowOrigins: allowedOrigins,
    allowMethods: ["GET", "PUT", "POST", "DELETE", "PATCH"],
    allowHeaders: [
      "Authorization",
      "Content-Type",
      "hx-current-url",
      "hx-request",
      "hx-trigger",
      "hx-target",
      "Location",
      "X-Recaptcha-Token",
    ],
    allowCredentials: true,
    maxAge: isLocal ? "1 minute" : "1 day",
    exposeHeaders: ["HX-Redirect", "hx-location", "hx-trigger"],
  },
});

const telegramWebhookFunction = new sst.aws.Function("TelegramWebhookFunction", {
  url: true,
  link: [secret.telegramBotToken, secret.telegramBotApiKey, secret.secretTursoUrl, receiptsBucket, secret.htmlToPdfFunction],
  handler: "packages/backend/kyo-repo/cmd/telegram-webhook/telegram-webhook.go",
  runtime: "go",
  permissions: [
    {
      actions: ["lambda:InvokeFunction"],
      resources: [secret.htmlToPdfFunction.value]
    },
  ]
})

const verifyAccessFunction = new sst.aws.Function("VerifyAccess", {
  link: [secret.appClientId, auth],
  handler: "packages/backend/openauthclient/verify.handler",
  // environment: {
  //   AUTH_SERVER_URL: `https://${authDomain}`,
  // },
});

// const htmlToPdf = new sst.aws.Function("HtmlToPdf", {
//   link: [receiptsBucket],
//   handler: "packages/backend/html-to-pdf/index.handler",
//   nodejs: {
//     install: ["@sparticuz/chromium"],
//   },
//   timeout: "80 seconds",
//   memory: "2 GB",
// });

const receiptPdfQueue = new sst.aws.Queue("ReceiptPdfQueue", {
  //not supported for S3 notificationsm
  fifo: {
    contentBasedDeduplication: true,
  },
  visibilityTimeout: "310 seconds",
  transform: {
    queue: args => {
      args.receiveWaitTimeSeconds = 20; // Long polling to reduce empty responses
    }
  }
});

receiptPdfQueue.subscribe({
  link: [
    secret.secretTursoUrl,
    receiptsBucket,
    secret.mailerConfigsSecret,
    secret.altEmailsRecipientSecret,
    secret.htmlToPdfFunction,
  ],
  environment: {
    SEND_MAIL: isLocal ? "false" : "true",
  },
  runtime: "go",
  handler: "packages/backend/kyo-repo/cmd/process-pdf-objects/process-pdf-objects.go",
  timeout: "300 seconds",
  permissions: [
    {
      actions: ["lambda:InvokeFunction"],
      resources: [secret.htmlToPdfFunction.value]
    },
  ],
});


let authRedirectUrl: Output<string> = "";
if ($dev) {
  // authRedirectUrl = api.url.apply(v => `${v}`);
  authRedirectUrl = `https://${domain}`;
  console.log(`AuthRedirectUrl ${authRedirectUrl}`)
}

const mainApiFunction = new sst.aws.Function("MainApiFunction", {
  handler: "packages/backend/kyo-repo/cmd/app/app.go",
  runtime: "go",
  link: [
    bcvBucket,
    secret.secretTursoUrl,
    secret.bcvUrl,
    secret.bcvFileStartPath,
    secret.appClientId,
    auth,
    verifyAccessFunction,
    receiptsBucket,
    receiptPdfQueue,
    secret.mailerConfigsSecret,
    secret.htmlToPdfFunction,
    webAssetsBucket,
    isrGenFunction,
    secret.telegramBotToken,
    secret.telegramBotApiKey,
    telegramWebhookFunction,
    secret.captchaSecretKey,
    secret.enableCaptcha,
    secret.posthogApiKey,
  ],
  environment: {
    ISR_PREFIX: isrPrefix,
    AUTH_SERVER_URL: authRedirectUrl,
  },
  memory: "2048 MB",
  timeout: "60 seconds",
  permissions: [
    {
      actions: ["lambda:InvokeFunction"],
      resources: [secret.htmlToPdfFunction.value]
    },
  ]
});

api.route("GET /api/{proxy+}", mainApiFunction.arn);
api.route("POST /api/{proxy+}", mainApiFunction.arn);
api.route("PUT /api/{proxy+}", mainApiFunction.arn);
api.route("DELETE /api/{proxy+}", mainApiFunction.arn);


export const site = new sst.aws.StaticSite("WebApp", {
  path: "packages/frontend/app",
  router: {
    instance: myRouter
  },
  environment: {
    // Accessible in the browser
    VITE_VAR_ENV: `https://${apiDomain}`,
    VITE_IS_DEV: isLocal.toString(),
    VITE_ISR_PREFIX: isrPrefix,
    VITE_RECAPTCHA_SITE_KEY: secret.captchaSiteKey.value,
    VITE_CAPTCHA_ENABLED: secret.enableCaptcha.value,
    VITE_POSTHOG_API_KEY: secret.posthogApiKey.value,
  },
  build: {
    command: "bun run build",
    output: "dist",
  },
  invalidation: {
    paths: "all",
    wait: $app.stage === PROD_STAGE,
  },
  assets: {
    bucket: webAssetsBucket.name,
    routes: ["isr", "assets"],
    fileOptions: [
      // {
      //   files: "index.html",
      //   cacheControl: "max-age=0,no-cache,must-revalidate,public"
      //   // cacheControl: "public,max-age=0,s-maxage=0,must-revalidate"
      //   // cacheControl: "max-age=0,no-cache,no-store,must-revalidate",
      // },
      {
        files: "isr/**/*",
        cacheControl: "max-age=0,no-cache,no-store,must-revalidate",
      },
      {
        files: ["**/*"],
        ignore: [
            // "index.html",
          "isr/**/*"],
        cacheControl: "public,max-age=31536000,immutable",
      },
      // {
      //   files: "**/*.html",
      //   cacheControl: "max-age=0,no-cache,no-store,must-revalidate"
      // }
    ],
  },
  transform: {
    cdn: (args) => {

      args.transform = {
        distribution: (disArgs) => {
          disArgs.httpVersion = "http2and3";
        }

      }
    }
  }
});

//
// export const kyoBot = new sst.aws.StaticSite("KyoBotWebApp", {
//   path: "packages/frontend/kyo-bot",
//   router: {
//     instance: myRouter,
//     domain: subdomain("kyo-bot"),
//   },
//   environment: {
//     // Accessible in the browser
//     VITE_VAR_ENV: `https://${apiDomain}`,
//     VITE_IS_DEV: isLocal.toString(),
//   },
//   build: {
//     command: "bun run build",
//     output: "dist",
//   },
//   assets: {
//     fileOptions: [
//       {
//         files: "index.html",
//         cacheControl: "max-age=0,no-cache,must-revalidate,public"
//       },
//       {
//         files: ["**/*"],
//         ignore: ["index.html", "isr/**/*"],
//         cacheControl: "public,max-age=31536000,immutable",
//       },
//     ],
//   },
//   transform: {
//     cdn: (args) => {
//
//       args.transform = {
//         distribution: (disArgs) => {
//           disArgs.httpVersion = "http2and3";
//         }
//
//       }
//     }
//   },
// });

console.log(`AuthServer URL: ${auth.url}`);


const authClientFunction = new sst.aws.Function("AuthClient", {
  url: true,
  link: [secret.appClientId, auth, site, secret.posthogApiKey],
  handler: "packages/backend/openauthclient/index.handler",
  environment: {
    IS_LOCAL: isLocal.toString(),
    // AUTH_SERVER_URL: authRedirectUrl,
    API_URL: `https://${apiDomain}`,
  },
});


api.route("GET /authorize", authClientFunction.arn);
api.route("GET /callback", authClientFunction.arn);
api.route("GET /", authClientFunction.arn);


myRouter.route("/api/authorize", authClientFunction.url, {
  rewrite: {
    regex: "^/api/(.*)$",
    to: "/$1"
  }
});

myRouter.route("/api/callback", authClientFunction.url, {
  rewrite: {
    regex: "^/api/(.*)$",
    to: "/$1"
  }
});


myRouter.route("/api", api.url);