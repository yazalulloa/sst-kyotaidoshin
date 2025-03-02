import {Resource} from "sst";
import {createClient} from "@openauthjs/openauth/client"
import type {Handler} from "aws-lambda";
import {subjects} from "@kyotaidoshin/auth/subjects";


const client = createClient({
  clientID: Resource.AppClientId.value,
  issuer: Resource.AuthServer.url
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

  const verified = await client.verify(subjects, accessToken!, {
    refresh: refreshToken,
  })

  if (verified.err) throw new Error("Invalid access token")

  return verified.subject;
}