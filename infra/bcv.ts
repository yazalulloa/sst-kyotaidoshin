import {secret} from "./secrets";
import {bcvBucket, webAssetsBucket} from "./storage";

export const bcvFunction = new sst.aws.Function("BcvFunction", {
  url: true,
  link: [
    bcvBucket,
    secret.bcvUrl,
    secret.bcvFileStartPath,
    secret.secretTursoUrl,
    secret.telegramBotToken,
    secret.telegramBotApiKey,
  ],
  runtime: "go",
  handler: "packages/backend/kyo-repo/cmd/bcv/bcv.go",
  timeout: "90 seconds",
});
// https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-scheduled-rule-pattern.html
export const bcvCron = new sst.aws.Cron("bcv-cron", {
  schedule: "cron(0/15 18-23 ? * MON-FRI *)",
  // schedule: "cron(0/15 * * * ? *)",
  function: bcvFunction.arn,
});


export const bcvQueue = new sst.aws.Queue("BcvQueue", {
  //not supported for S3 notifications
  fifo: false,
  visibilityTimeout: "300 seconds",
  transform: {
    queue: args => {
      args.receiveWaitTimeSeconds = 20; // Long polling to reduce empty responses
    }

  }
});

export const processBcvFileFunction = new sst.aws.Function("ProcessBcvFile", {
  link: [
    secret.secretTursoUrl,
    bcvBucket,
    bcvQueue,
    webAssetsBucket,
    secret.telegramBotToken,
    secret.telegramBotApiKey,
  ],
  runtime: "go",
  handler: "packages/backend/kyo-repo/cmd/process-bcv-file/process-bcv-file.go",
  timeout: "90 seconds",
  // volume: {
  //   efs: efs,
  //   path: "/mnt/efs"
  // }
  // permissions: [
  //   {
  //     actions: ["s3:*"],c
  //   }
  // ]
});
bcvQueue.subscribe(processBcvFileFunction.arn, {
  batch: {
    window: "20 seconds"
  },
  transform: {
    eventSourceMapping: args => {
      args.maximumBatchingWindowInSeconds = 20; // Reduce latency for processing
    }
  }
});
bcvBucket.notify({
  notifications: [
    {
      name: "ProcessFileSubscriber",
      queue: bcvQueue,
      events: ["s3:ObjectCreated:Put", "s3:ObjectCreated:Post"],
    },
  ],
});