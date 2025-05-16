import {isLocal, PROD_STAGE} from "./util";


export const stageDomain = $app.stage === PROD_STAGE ? "" : `${$app.stage}.`;
export const currentWebUrl = `${stageDomain}${process.env.DOMAIN}`
console.log("currentWebUrl", currentWebUrl);

export const apiDomain = `api.${currentWebUrl}`

export const allowedOrigins = isLocal ? ["http://localhost:5173"] : [`https://${currentWebUrl}`];

export const authDomain = `auth.${currentWebUrl}`
console.log('AuthDomain', authDomain)