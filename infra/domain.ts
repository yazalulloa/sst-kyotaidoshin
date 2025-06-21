import {isLocal, PROD_STAGE, DEV_STAGE} from "./util";


// export const stageDomain = $app.stage === PROD_STAGE ? "" : `${$app.stage}.`;
// export const currentWebUrl = `${stageDomain}${process.env.DOMAIN}`
// console.log("currentWebUrl", currentWebUrl);

const isPermanentStage = [PROD_STAGE, DEV_STAGE].includes($app.stage);

export const domain = $app.stage === PROD_STAGE
    ? process.env.DOMAIN
    : $app.stage === DEV_STAGE
        ? `${$app.stage}.${process.env.DOMAIN}`
        : `${$app.stage}.dev.${process.env.DOMAIN}`;

console.log('Domain', domain);

function subdomain(name: string) {
  if (isPermanentStage) return `${name}.${domain}`;
  return `${name}-${domain}`;
}

// export const apiDomain = subdomain("api")
export const apiDomain = `${domain}/api`;

export const allowedOrigins = isLocal ? ["http://localhost:5173"] : [`https://${domain}`];

// export const authDomain = `${domain}/auth`;
export const authDomain = subdomain("auth")
// console.log('AuthDomain', authDomain)

// export const myRouter = new sst.aws.Router("MyRouter", {
//   domain: {
//     name: domain,
//     aliases: [`*.${domain}`],
//     dns: sst.aws.dns({override: true}),
//   }
// })

export const myRouter = isPermanentStage
    ? new sst.aws.Router("MyRouter", {
      domain: {
        name: domain,
        aliases: [`*.${domain}`],
        dns: sst.aws.dns({override: true}),
      }
    })
    : sst.aws.Router.get("MyRouter", process.env.DISTRIBUTION_ID);