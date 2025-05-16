export const secret = {
  secretTursoUrl: new sst.Secret("SecretTursoUrl"),
  bcvUrl: new sst.Secret("SecretBcvUrl"),
  bcvFileStartPath: new sst.Secret("SecretBcvFileStartPath"),

  appClientId: new sst.Secret("AppClientId"),
  githubClientId: new sst.Secret("GithubClientId"),
  githubClientSecret: new sst.Secret("GithubClientSecret"),
  googleClientId: new sst.Secret("GoogleClientId"),
  googleClientSecret: new sst.Secret("GoogleClientSecret"),
  mailerConfigsSecret: new sst.Secret("MailerConfigs"),
  altEmailsRecipientSecret: new sst.Secret("AltEmailsRecipient", ""),
  htmlToPdfFunction: new sst.Secret("HtmlToPdfFunction"),
  telegramBotToken: new sst.Secret("TelegramBotToken"),
  telegramBotApiKey: new sst.Secret("TelegramBotApiKey"),
  captchaSiteKey: new sst.Secret("CaptchaSiteKey"),
  captchaSecretKey: new sst.Secret("CaptchaSecretKey"),
}

export const allSecrets = [...Object.values(secret)];