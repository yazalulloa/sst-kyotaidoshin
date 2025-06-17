import {handle} from "hono/aws-lambda"
import {issuer} from "@openauthjs/openauth"
import {GithubProvider} from "@openauthjs/openauth/provider/github"
import {subjects} from "./subjects"
import {GoogleProvider} from "@openauthjs/openauth/provider/google";
import {Resource} from "sst";
import {Select} from "@openauthjs/openauth/ui/select"
import type {Theme} from "@openauthjs/openauth/ui/theme"
import {InvokeCommand, LambdaClient} from "@aws-sdk/client-lambda";
import {DynamoStorage} from "@openauthjs/openauth/storage/dynamo"
import { PostHog } from 'posthog-node'

const lambda = new LambdaClient({});

const posthog = new PostHog(Resource.PosthogApiKey.value, { host: 'https://us.i.posthog.com' })

const storage = DynamoStorage({
  table: "kyotaidoshin-tokens",
  pk: "pk",
  sk: "sk"
})

const MY_THEME: Theme = {
  title: "Kyotaidoshin",
  radius: "md",
  favicon: "https://kyotaidoshin.com/favicon.ico",
  primary: {
    light: "#FFF",
    dark: "#000"
  },
  background: {
    light: "#FFF",
    dark: "#000"
  },
  logo: "https://kyotaidoshin.com/favicon.ico",
  // ...
}

const app = issuer({
  subjects,
  storage: storage,
  // // Remove after setting custom domain
  // allow: async (input, req) => {
  //   console.log("Allow: ", input, req)
  //   return true
  // },
  providers: {
    github: GithubProvider({
      clientID: Resource.GithubClientId.value,
      clientSecret: Resource.GithubClientSecret.value,
      scopes: ["read:user", "user:email"],
    }),
    google: GoogleProvider({
      clientID: Resource.GoogleClientId.value,
      clientSecret: Resource.GoogleClientSecret.value,
      scopes: ["openid", "profile", "email"],
    })
  },
  theme: MY_THEME,
  select: Select({
    providers: {
      github: {display: "GitHub"},
      google: {display: "Google"}
    }
  }),
  success: async (ctx, value) => {
    let json = JSON.stringify(value)

    console.log("Value: ", json)
    let output = await lambda.send(
        new InvokeCommand({
          FunctionName: Resource.ProcessUser.name,
          InvocationType: "RequestResponse",
          Payload: json,
        })
    );


    let payload = output.Payload?.transformToString("utf-8")
    if (output.StatusCode === 200 && payload) {
      const jsonObject = JSON.parse(payload);

      return ctx.subject("user", {
        userID: jsonObject.userId,
        workspaceID: jsonObject.workspaceId,
      })
    }

    let msgErr = payload ?? "Empty payload"

    throw new Error(`Some error ${output.StatusCode}: ${msgErr}`)
  }

})


app.onError((err, c) => {


  console.error("Error in AuthServer:", err, c)

  posthog.captureException(new Error(err.message, { cause: err }), undefined, {
    path: c.req.path,
    method: c.req.method,
    url: c.req.url,
    headers: c.req.header(),
    // ... other properties
  })
  // posthog.shutdown()
  // other error handling logic
  return c.text('Internal Server Error', 500)
})

export const handler = handle(app)
