import {handle} from "hono/aws-lambda"
import {issuer} from "@openauthjs/openauth"
import {GithubProvider} from "@openauthjs/openauth/provider/github"
import {subjects} from "./subjects"
import {GoogleProvider} from "@openauthjs/openauth/provider/google";
import {Resource} from "sst";
import {Select} from "@openauthjs/openauth/ui/select"
import {THEME_OPENAUTH} from "@openauthjs/openauth/ui/theme"
import {InvokeCommand, LambdaClient} from "@aws-sdk/client-lambda";
import {DynamoStorage} from "@openauthjs/openauth/storage/dynamo"
import {v4 as uuidv4} from 'uuid';

const lambda = new LambdaClient({});

const storage = DynamoStorage({
  table: "kyotaidoshin-tokens",
  pk: "pk",
  sk: "sk"
})

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
      // scopes: ["openid", "profile", "email"],
      scopes: ["openid", "user:email"],
      pkce: true,
      query: {
        nonce: uuidv4(),
      }
    }),
    google: GoogleProvider({
      clientID: Resource.GoogleClientId.value,
      clientSecret: Resource.GoogleClientSecret.value,
      scopes: ["openid", "profile", "email"],
    })
  },
  theme: THEME_OPENAUTH,
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


export const handler = handle(app)
