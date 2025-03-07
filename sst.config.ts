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
    };
  },
  async run() {

    let isLocal = Boolean($app.stage !== PROD_STAGE && $app.stage !== STAG_STAGE);
    console.log("isLocal", isLocal);
    const bucket = new sst.aws.Bucket("bcv-bucket", {
      versioning: false,
    });
    const secretTursoUrl = new sst.Secret("SecretTursoUrl");
    const bcvUrl = new sst.Secret("SecretBcvUrl");
    const bcvFileStartPath = new sst.Secret("SecretBcvFileStartPath");
    const webUrl = new sst.Secret("WebUrl");
    const appClientId = new sst.Secret("AppClientId")

    const githubClientId = new sst.Secret("GithubClientId");
    const githubClientSecret = new sst.Secret("GithubClientSecret");
    const googleClientId = new sst.Secret("GoogleClientId");
    const googleClientSecret = new sst.Secret("GoogleClientSecret");
    // const domain = new sst.Secret("Domain");

    const processUserFunction = new sst.aws.Function("ProcessUser", {
      link: [secretTursoUrl],
      handler: "packages/backend/process-user/",
      runtime: "go",
    });


    const auth = new sst.aws.Auth("AuthServer", {
      forceUpgrade: "v2",
      issuer: {
        handler: "packages/auth/index.handler",
        link: [githubClientId, githubClientSecret, googleClientId, googleClientSecret, processUserFunction],
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


    // const processRatesQueue = new sst.aws.Queue("ProcessRatesQueue", {
    //   //not supported for S3 notificationsm
    //   fifo: true,
    // });
    //
    // const processRatesFunction = new sst.aws.Function("ProcessRates", {
    //   link: [secretTursoUrl, processRatesQueue],
    //   runtime: "go",
    //   handler: "packages/backend/process-rates/",
    // });
    //
    //
    // processRatesQueue.subscribe(processRatesFunction.arn)


    const processBcvFileFunction = new sst.aws.Function("ProcessBcvFile", {
      link: [secretTursoUrl, bucket, bcvQueue],
      runtime: "go",
      handler: "packages/backend/process-bcv-file/",
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

    // queue.subscribe({
    //   name: "process-file-go",
    //   runtime: "go",
    //   handler: "packages/backend/process-bcv-file/",
    //   // name: "process-file-js",
    //   // handler: "subscriber.handler",
    // })
    bcvQueue.subscribe(processBcvFileFunction.arn);

    // const subscriber = new sst.aws.Function("MyFunction", {
    //   handler: "subscriber.handler"
    // });
    //
    // const queue = new sst.aws.Queue("MyQueue");
    // queue.subscribe(subscriber.arn);

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
      handler: "packages/backend/bcv/",
      timeout: "90 seconds",
    });

    // https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-scheduled-rule-pattern.html
    new sst.aws.Cron("bcv-cron", {
      schedule: "cron(0/15 18-23 ? * MON-FRI *)",
      // schedule: "cron(0/15 * * * ? *)",
      function: bcvFunction.arn,
    })

    // let allowedOrigins = isLocal ? ["*"] : [webUrl.value];
    let allowedOrigins = isLocal ? ["http://localhost:5173"] : [webUrl.value];

    const api = new sst.aws.ApiGatewayV2("API", {
      link: [auth],
      // domain: domain.value,
      cors: {
        allowOrigins: allowedOrigins,
        allowMethods: ["GET", "PUT", "POST", "DELETE"],
        allowHeaders: ["Authorization", "Content-Type", "hx-current-url", "hx-request", "hx-trigger", "hx-target"],
        allowCredentials: true,
        maxAge: isLocal ? "1 minute" : "1 day",
        exposeHeaders: ["HX-Redirect", "hx-location"],
      }
    });

    const uploadBackupBucket = new sst.aws.Bucket("UploadBackupBucket", {
      versioning: false,
      cors: {
        allowHeaders: ["Content-Type", "hx-current-url", "hx-request", "hx-trigger", "hx-target"],
        allowOrigins: allowedOrigins,
        allowMethods: ["GET", "POST", "PUT"],
        exposeHeaders: [],
        maxAge: isLocal ? "1 minute" : "1 day",
      }
    });

    const uploadBackupQueue = new sst.aws.Queue("UploadBackupQueue", {
      //not supported for S3 notificationsm
      fifo: false,
      visibilityTimeout: "300 seconds",
    });

    uploadBackupQueue.subscribe({
      link: [secretTursoUrl, uploadBackupBucket, uploadBackupQueue],
      runtime: "go",
      handler: "packages/backend/process-backup/",
      timeout: "90 seconds",
    });

    uploadBackupBucket.notify({
      notifications: [
        {
          name: "ProcessBackupSubscriber",
          queue: uploadBackupQueue,
          events: [ "s3:ObjectCreated:Post"],
        },
      ],
    });

    const verifyAccessFunction = new sst.aws.Function("VerifyAccess", {
      link: [appClientId, auth],
      handler: "packages/backend/openauthclient/verify.handler",
    });

    const receiptsBucket = new sst.aws.Bucket("ReceiptsBucket", {
      versioning: false,
    });

    const htmlToPdf = new sst.aws.Function("HtmlToPdf", {
      link: [receiptsBucket],
      handler: "packages/backend/html-to-pdf/index.handler",
      nodejs: {
        install: ["@sparticuz/chromium"],
      },
      timeout: "80 seconds",
      memory: "2 GB",
    });

    const mainApiFunction = new sst.aws.Function("MainApiFunction", {
      handler: "packages/backend/api",
      runtime: "go",
      link: [bucket, secretTursoUrl, bcvUrl, bcvFileStartPath, uploadBackupBucket, appClientId, auth, verifyAccessFunction, receiptsBucket, htmlToPdf],
      timeout: "60 seconds",
    });

    api.route("GET /api/{proxy+}", mainApiFunction.arn);
    api.route("POST /api/{proxy+}", mainApiFunction.arn);
    api.route("PUT /api/{proxy+}", mainApiFunction.arn);
    api.route("DELETE /api/{proxy+}", mainApiFunction.arn);

    // api.route("$default", {
    //   handler: "packages/backend/api",
    //   runtime: "go",
    //   link: [bucket, secretTursoUrl, bcvUrl, bcvFileStartPath, uploadBackupBucket, apiFunction],
    //   timeout: "60 seconds",
    // });

    const site = new sst.aws.StaticSite("WebApp", {
      path: "packages/frontend/app",
      environment: {
        // Accessible in the browser
        VITE_VAR_ENV: api.url,
        VITE_IS_DEV: isLocal.toString(),
      },
      build: {
        command: "bun run build",
        output: "dist"
      },
      assets: {
        fileOptions: [
          {
            files: ["**/*"],
            cacheControl: "max-age=21600,must-revalidate,public,immutable"
          },
          // {
          //   files: "**/*.html",
          //   cacheControl: "max-age=0,no-cache,no-store,must-revalidate"
          // }
        ]
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
      handler: "packages/backend/openauthclient/index.handler",
      environment: {
        IS_LOCAL: isLocal.toString(),
      }
    });


    api.route("GET /authorize", authClientFunction.arn);
    api.route("GET /callback", authClientFunction.arn);
    api.route("GET /", authClientFunction.arn);

    return {
      Web_App: webUrl.value,
      SiteUrl: site.url,
      VerifyAccess: verifyAccessFunction.arn,
      // ApiFunction: apiFunction.url,
      // MyBucket: bucket.name,
      // BcvUrl: bcvUrl.value,
      // BcvFileStartPath: bcvFileStartPath.value,
      // SiteUrl: site.url,
      // url: router.url,
    };
  },
});
