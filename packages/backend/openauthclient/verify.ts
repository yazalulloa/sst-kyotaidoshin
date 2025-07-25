import {Resource} from "sst";
import {createClient} from "@openauthjs/openauth/client"
import type {Handler} from "aws-lambda";
import {subjects} from "@kyotaidoshin/auth/subjects";


const client = createClient({
  clientID: Resource.AppClientId.value,
  issuer: Resource.AuthServer.url,
  // issuer: process.env.AUTH_SERVER_URL,
})


export const handler: Handler = async (event, context) => {
  let accessToken: string | undefined = event?.accessToken?.trim()
  if (!accessToken || accessToken.length == 0) {
    return {
      statusCode: 400,
      body: "Access token is required",
    }
  }

  let refreshToken: string | undefined = event?.refreshToken?.trim()

  let verified;

  try {
    verified = await client.verify(subjects, accessToken!, {
      refresh: refreshToken,
    })
  } catch (e) {
    console.error(`Access token  ${accessToken}`)
    console.error(`Refresh token ${refreshToken}`)
    console.error("Error client.verify", e)
    return {
      statusCode: 401,
      body: "Invalid access token",
    }
  }

  if (verified.err) {
    console.error(`Access token  ${accessToken}`)
    console.error(`Refresh token ${refreshToken}`)
    console.error("Error verifying token", verified.err)
    throw new Error("Invalid access token")
  }

  return verified.subject;
}