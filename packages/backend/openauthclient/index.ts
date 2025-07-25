import {type Context, Hono} from "hono"
import {getCookie, setCookie} from "hono/cookie"
import {createClient} from "@openauthjs/openauth/client"
import {handle} from "hono/aws-lambda"
import {subjects} from "../../auth/subjects"
import {Resource} from "sst";
import { PostHog } from 'posthog-node'
import * as process from "node:process";

const client = createClient({
  clientID: Resource.AppClientId.value,
  issuer: Resource.AuthServer.url,
  // issuer: process.env.AUTH_SERVER_URL,
})


const posthog = new PostHog(Resource.PosthogApiKey.value, { host: 'https://us.i.posthog.com' })

const isLocal = getIsLocal();
const apiUrl = process.env.API_URL

function getIsLocal(): boolean {

  if (process.env.IS_LOCAL && process.env.IS_LOCAL === "true") {
    return true
  }

  return false
}

const redirectUrl = (isLocal ? "http://localhost:5173" : Resource.WebApp.url) + "/logged_in"

const app = new Hono()
.get("/authorize", async (c) => {
  // const origin = new URL(c.req.url).origin
  const {url} = await client.authorize(apiUrl + "/callback", "code")
  return c.redirect(url, 302)
})
.get("/callback", async (c) => {
  // const origin = new URL(c.req.url).origin
  try {
    const code = c.req.query("code")
    if (!code) throw new Error("Missing code")
    const exchanged = await client.exchange(code, apiUrl + "/callback")
    if (exchanged.err)
      return new Response(exchanged.err.toString(), {
        status: 400,
      })
    setSession(c, exchanged.tokens.access, exchanged.tokens.refresh)

    return c.redirect(redirectUrl, 302)
    // return c.redirect("/", 302)
  } catch (e: any) {
    return new Response(e.toString())
  }
})
.get("/", async (c) => {

  const access = getCookie(c, "access_token")
  const refresh = getCookie(c, "refresh_token")

  if (!access) return c.redirect("/authorize", 302)

  try {
    const verified = await client.verify(subjects, access!, {
      refresh,
    })


    if (verified.err) throw new Error("Invalid access token")
    if (verified.tokens)
      setSession(c, verified.tokens.access, verified.tokens.refresh)


    return c.redirect(redirectUrl, 302)
    // return c.json(verified.subject)
  } catch (e) {
    console.error(e)
    return c.redirect("/authorize", 302)
  }
})

app.onError((err, c) => {

  console.error("Error in OpenAuthClient:", err, c)


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

function setSession(c: Context, accessToken?: string, refreshToken?: string) {

  const sameSite = isLocal ? "none" : "strict"

  if (accessToken) {
    setCookie(c, "access_token", accessToken, {
      httpOnly: true,
      secure: true,
      sameSite: sameSite,
      path: "/",
      maxAge: 34560000,
    })
  }
  if (refreshToken) {
    setCookie(c, "refresh_token", refreshToken, {
      httpOnly: true,
      secure: true,
      sameSite: sameSite,
      path: "/",
      maxAge: 34560000,
    })
  }
}
