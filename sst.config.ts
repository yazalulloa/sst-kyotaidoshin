/// <reference path="./.sst/platform/config.d.ts" />

const PROD_STAGE = "production";

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

    const bucket = new sst.aws.Bucket("bcv-bucket", {
      versioning: false,
    });
    const secretTursoUrl = new sst.Secret("SecretTursoUrl");
    const bcvUrl = new sst.Secret("SecretBcvUrl");
    const bcvFileStartPath = new sst.Secret("SecretBcvFileStartPath");
    // const domain = new sst.Secret("Domain");


    const queue = new sst.aws.Queue("BcvQueue", {
      //not supported for S3 notificationsm
      fifo: false,
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
      link: [secretTursoUrl, bucket, queue],
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
    queue.subscribe(processBcvFileFunction.arn);

    // const subscriber = new sst.aws.Function("MyFunction", {
    //   handler: "subscriber.handler"
    // });
    //
    // const queue = new sst.aws.Queue("MyQueue");
    // queue.subscribe(subscriber.arn);

    bucket.notify({
      notifications: [
        {
          // function: {
          //   runtime: "go",
          //   handler: "packages/backend/process-bcv-file/",
          // },
          name: "ProcessFileSubscriber",
          queue: queue,
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

    const api = new sst.aws.ApiGatewayV2("API", {
      // domain: domain.value,
      cors: {
        allowOrigins: ["*"],
        allowMethods: ["GET", "HEAD", "PUT", "POST", "DELETE", "OPTIONS"],
        // allowHeaders: ["date", "keep-alive", "access-control-request-headers"],
        // exposeHeaders: ["date", "keep-alive", "access-control-request-headers"]
      }
    });

    const uploadBackupBucket = new sst.aws.Bucket("UploadBackup", {
      versioning: false,
      cors: {
        allowHeaders: ["*"],
        allowOrigins: ["*"],
        allowMethods: ["DELETE", "GET", "HEAD", "POST", "PUT"],
        exposeHeaders: [],
        maxAge: "0 seconds"
      }
    });

    const apiFunction = new sst.aws.Function("ApiFunction", {
      url: true,
      handler: "packages/backend/api",
      runtime: "go",
      link: [bucket, secretTursoUrl, bcvUrl, bcvFileStartPath, uploadBackupBucket],
      timeout: "60 seconds",
    });


    api.route("$default", {
      handler: "packages/backend/api",
      runtime: "go",
      link: [bucket, secretTursoUrl, bcvUrl, bcvFileStartPath, uploadBackupBucket, apiFunction],
      timeout: "60 seconds",
    });

    // const api = new sst.aws.Function("ApiFunction", {
    //   url: true,
    //   handler: "packages/backend/api",
    //   runtime: "go",
    //   link: [bucket, secretTursoUrl]
    // })

    const site = new sst.aws.StaticSite("WebApp", {
      path: "packages/frontend/app",
      environment: {
        // Accessible in the browser
        VITE_VAR_ENV: api.url,
        VITE_IS_DEV: Boolean($app.stage !== PROD_STAGE).toString(),
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

    return {
      ApiFunction: apiFunction.url,
      MyBucket: bucket.name,
      BcvUrl: bcvUrl.value,
      BcvFileStartPath: bcvFileStartPath.value,
      SiteUrl: site.url,
      // url: router.url,
    };
  },
});
