import {secret} from "./secrets";

export const bcvFunction = new sst.aws.Function("BcvFunction", {
  url: true,
  link: [
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
