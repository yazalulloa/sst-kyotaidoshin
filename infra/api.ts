import {secret} from "./secrets";
import {allowedOrigins, apiDomain, authDomain, currentWebUrl} from "./domain";
import {bcvBucket, receiptsBucket, webAssetsBucket} from "./storage";
import {isLocal, isrPrefix} from "./util";


const processUserFunction = new sst.aws.Function("ProcessUser", {
  link: [secret.secretTursoUrl, secret.telegramBotToken, secret.telegramBotApiKey],
  handler: "packages/backend/kyo-repo/cmd/process-user/process-user.go",
  runtime: "go",
});


const auth = new sst.aws.Auth("AuthServer", {
  domain: {
    name: authDomain
  },
  forceUpgrade: "v2",
  issuer: {
    handler: "packages/auth/index.handler",
    link: [
      secret.githubClientId,
      secret.githubClientSecret,
      secret.googleClientId,
      secret.googleClientSecret,
      processUserFunction,
    ],
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
  domain: {
    name: apiDomain
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
  link: [secret.telegramBotToken, secret.telegramBotApiKey, secret.secretTursoUrl],
  handler: "packages/backend/kyo-repo/cmd/telegram-webhook/telegram-webhook.go",
  runtime: "go",
})

const verifyAccessFunction = new sst.aws.Function("VerifyAccess", {
  link: [secret.appClientId, auth],
  handler: "packages/backend/openauthclient/verify.handler",
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
  ]
});

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
    secret.posthogApiKey,
  ],
  environment: {
    ISR_PREFIX: isrPrefix
  },
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
  domain: {
    name: currentWebUrl
  },
  environment: {
    // Accessible in the browser
    VITE_VAR_ENV: `https://${apiDomain}`,
    VITE_IS_DEV: isLocal.toString(),
    VITE_ISR_PREFIX: isrPrefix,
    VITE_RECAPTCHA_SITE_KEY: secret.captchaSiteKey.value,
  },
  build: {
    command: "bun run build",
    output: "dist",
  },
  assets: {
    bucket: webAssetsBucket.name,
    routes: ["isr", "assets"],
    fileOptions: [
      {
        files: "index.html",
        cacheControl: "public,max-age=0,s-maxage=0,must-revalidate"
        // cacheControl: "max-age=0,no-cache,no-store,must-revalidate",
      },
      {
        files: "isr/**/*",
        cacheControl: "max-age=0,no-cache,no-store,must-revalidate",
      },
      {
        files: ["**/*"],
        ignore: ["index.html", "isr/**/*"],
        cacheControl: "public,max-age=21600,immutable",
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


// const router = new sst.aws.Router("MyRouter", {
//   routes: {
//     "/api/*": api.url,
//     "/*": site.url,
//   },
// });
const authClientFunction = new sst.aws.Function("AuthClient", {
  link: [secret.appClientId, auth, site],
  handler: "packages/backend/openauthclient/index.handler",
  environment: {
    IS_LOCAL: isLocal.toString(),
  },
});
api.route("GET /authorize", authClientFunction.arn);
api.route("GET /callback", authClientFunction.arn);
api.route("GET /", authClientFunction.arn);