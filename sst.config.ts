/// <reference path="./.sst/platform/config.d.ts" />
const PROD_STAGE: string = "production";
const STAG_STAGE: string = "staging";
export default $config({
  app(input) {
    return {
      name: "kyotaidoshin",
      removal: input?.stage === PROD_STAGE ? "retain" : "remove",
      protect: [PROD_STAGE].includes(input?.stage),
      home: "aws",
      providers: {aws: "6.71.0"},
    };
  },
  async run() {
    let isLocal = Boolean(
        $app.stage !== PROD_STAGE && $app.stage !== STAG_STAGE,
    );
    console.log("isLocal", isLocal);

    const domain = process.env.DOMAIN;
    const stageDomain = $app.stage === PROD_STAGE ? "" : `${$app.stage}.`;
    const currentWebUrl = `${stageDomain}${domain}`
    console.log("currentWebUrl", currentWebUrl);

    const bucket = new sst.aws.Bucket("bcv-bucket", {
      versioning: false,
    });
    const secretTursoUrl = new sst.Secret("SecretTursoUrl");
    const bcvUrl = new sst.Secret("SecretBcvUrl");
    const bcvFileStartPath = new sst.Secret("SecretBcvFileStartPath");

    const appClientId = new sst.Secret("AppClientId");
    const githubClientId = new sst.Secret("GithubClientId");
    const githubClientSecret = new sst.Secret("GithubClientSecret");
    const googleClientId = new sst.Secret("GoogleClientId");
    const googleClientSecret = new sst.Secret("GoogleClientSecret");
    const mailerConfigsSecret = new sst.Secret("MailerConfigs");
    const altEmailsRecipientSecret = new sst.Secret("AltEmailsRecipient", "");
    const htmlToPdfFunction = new sst.Secret("HtmlToPdfFunction")
    const telegramBotToken = new sst.Secret("TelegramBotToken")
    const telegramBotApiKey = new sst.Secret("TelegramBotApiKey")
    const captchaSiteKey = new sst.Secret("CaptchaSiteKey")
    const captchaSecretKey = new sst.Secret("CaptchaSecretKey")

    const processUserFunction = new sst.aws.Function("ProcessUser", {
      link: [secretTursoUrl, telegramBotToken, telegramBotApiKey],
      handler: "packages/backend/kyo-repo/cmd/process-user/",
      runtime: "go",
    });


    const authDomain = `auth.${currentWebUrl}`
    console.log('AuthDomain', authDomain)
    const auth = new sst.aws.Auth("AuthServer", {
      domain: {
        name: authDomain
      },
      forceUpgrade: "v2",
      issuer: {
        handler: "packages/auth/index.handler",
        link: [
          githubClientId,
          githubClientSecret,
          googleClientId,
          googleClientSecret,
          processUserFunction,
        ],
      },
    });
    const bcvQueue = new sst.aws.Queue("BcvQueue", {
      //not supported for S3 notificationsm
      fifo: false,
      visibilityTimeout: "300 seconds",
    });
    // const vpc = new sst.aws.Vpc("MyVpc", {
    //   nat: {
    //     ec2: {
    //       instance: "t4g.nano"
    //     }
    //   }
    // });
    // const efs = new sst.aws.Efs("MyEfs", {
    //   vpc: vpc
    // });

    const webAssetsBucket = new sst.aws.Bucket("WebAssetsBucket", {
      access: "cloudfront",
    });

    const isrPrefix = "isr/v6"

    const isrGenFunction = new sst.aws.Function("IsrGenFunction", {
      url: true,
      link: [webAssetsBucket, secretTursoUrl],
      environment: {
        ISR_PREFIX: isrPrefix
      },
      handler: "packages/backend/kyo-repo/cmd/isr-gen/",
      runtime: "go",
    });

    const processBcvFileFunction = new sst.aws.Function("ProcessBcvFile", {
      link: [secretTursoUrl, bucket, bcvQueue, webAssetsBucket],
      runtime: "go",
      handler: "packages/backend/kyo-repo/cmd/process-bcv-file/",
      timeout: "90 seconds",
      // volume: {
      //   efs: efs,
      //   path: "/mnt/efs"
      // }
      // permissions: [
      //   {
      //     actions: ["s3:*"],
      //     resources: [bucket.arn]
      //   }
      // ]
    });
    bcvQueue.subscribe(processBcvFileFunction.arn);
    bucket.notify({
      notifications: [
        {
          name: "ProcessFileSubscriber",
          queue: bcvQueue,
          events: ["s3:ObjectCreated:Put", "s3:ObjectCreated:Post"],
        },
      ],
    });
    const bcvFunction = new sst.aws.Function("BcvFunction", {
      url: true,
      link: [bucket, bcvUrl, bcvFileStartPath],
      runtime: "go",
      handler: "packages/backend/kyo-repo/cmd/bcv/",
      timeout: "90 seconds",
    });
    // https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-scheduled-rule-pattern.html
    new sst.aws.Cron("bcv-cron", {
      schedule: "cron(0/15 18-23 ? * MON-FRI *)",
      // schedule: "cron(0/15 * * * ? *)",
      function: bcvFunction.arn,
    });
    const apiDomain = `api.${currentWebUrl}`


    // let allowedOrigins = isLocal ? ["*"] : [webUrl.value];
//  uploadBackupBucket.domain.apply(v => `https://${v}`)
    const allowedOrigins = isLocal ? ["http://localhost:5173"] : [`https://${currentWebUrl}`];

    const api = new sst.aws.ApiGatewayV2("API", {
      link: [auth],
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
      link: [telegramBotToken, telegramBotApiKey, secretTursoUrl],
      handler: "packages/backend/kyo-repo/cmd/telegram-webhook/",
      runtime: "go",
    })

    const verifyAccessFunction = new sst.aws.Function("VerifyAccess", {
      link: [appClientId, auth],
      handler: "packages/backend/kyo-repo/cmd/openauthclient/verify.handler",
    });
    const receiptsBucket = new sst.aws.Bucket("ReceiptsBucket", {
      versioning: false,
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
        secretTursoUrl,
        receiptsBucket,
        mailerConfigsSecret,
        altEmailsRecipientSecret,
        htmlToPdfFunction,
      ],
      environment: {
        SEND_MAIL: isLocal ? "false" : "true",
      },
      runtime: "go",
      handler: "packages/backend/kyo-repo/cmd/process-pdf-objects/",
      timeout: "300 seconds",
      permissions: [
        {
          actions: ["lambda:InvokeFunction"],
          resources: [htmlToPdfFunction.value]
        },
      ]
    });

    const mainApiFunction = new sst.aws.Function("MainApiFunction", {
      handler: "packages/backend/kyo-repo/cmd/app/",
      runtime: "go",
      link: [
        bucket,
        secretTursoUrl,
        bcvUrl,
        bcvFileStartPath,
        appClientId,
        auth,
        verifyAccessFunction,
        receiptsBucket,
        receiptPdfQueue,
        mailerConfigsSecret,
        htmlToPdfFunction,
        webAssetsBucket,
        isrGenFunction,
        telegramBotToken,
        telegramBotApiKey,
        telegramWebhookFunction,
        captchaSecretKey,
      ],
      environment: {
        ISR_PREFIX: isrPrefix
      },
      timeout: "60 seconds",
      permissions: [
        {
          actions: ["lambda:InvokeFunction"],
          resources: [htmlToPdfFunction.value]
        },
      ]
    });

    api.route("GET /api/{proxy+}", mainApiFunction.arn);
    api.route("POST /api/{proxy+}", mainApiFunction.arn);
    api.route("PUT /api/{proxy+}", mainApiFunction.arn);
    api.route("DELETE /api/{proxy+}", mainApiFunction.arn);


    const site = new sst.aws.StaticSite("WebApp", {
      path: "packages/frontend/app",
      domain: {
        name: currentWebUrl
      },
      environment: {
        // Accessible in the browser
        VITE_VAR_ENV: `https://${apiDomain}`,
        VITE_IS_DEV: isLocal.toString(),
        VITE_ISR_PREFIX: isrPrefix,
        VITE_RECAPTCHA_SITE_KEY: captchaSiteKey.value,
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
          // Modify CloudFront distribution arguments here

          // Add other CloudFront-specific configurations
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
      link: [appClientId, auth, site],
      handler: "packages/backend/kyo-repo/cmd/openauthclient/index.handler",
      environment: {
        IS_LOCAL: isLocal.toString(),
      },
    });
    api.route("GET /authorize", authClientFunction.arn);
    api.route("GET /callback", authClientFunction.arn);
    api.route("GET /", authClientFunction.arn);
    return {
      SiteUrl: site.url,
    };
  },
});
