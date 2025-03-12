import chromium from '@sparticuz/chromium'
import puppeteer, {Browser} from 'puppeteer-core'
import type {Handler} from "aws-lambda";
import {Resource} from 'sst'
import {PutObjectCommand, S3Client} from '@aws-sdk/client-s3'

// This is the path to the local Chromium binary
const YOUR_LOCAL_CHROMIUM_PATH =
    "/home/yaz/software/localChromium/chromium/linux-1428138/chrome-linux/chrome";

const s3 = new S3Client();

class PdfItem {
  objectKey: string
  html: string

  constructor(objectKey: string, html: string) {
    this.objectKey = objectKey
    this.html = html
  }
}

export const handler: Handler = async (event: PdfItem[], context) => {

  if (event.length == 0) {
    return {
      statusCode: 400,
      body: "No items to process",
    }
  }

  let browser: Browser | null = null
  try {
    // console.log('Launching browser')
    browser = await puppeteer.launch({
      args: chromium.args,
      defaultViewport: chromium.defaultViewport,
      executablePath: process.env.SST_DEV
          ? YOUR_LOCAL_CHROMIUM_PATH
          : await chromium.executablePath(),
      headless: chromium.headless,
      // acceptInsecureCerts: true,
    })

    // console.log('Browser launched')


    const promises = [];

    for (let i = 0; i < event.length; i++) {
      const item = event[i]
      const html = decodeBase64UrlStr(item.html)
      const key = item.objectKey

      const promise = new Promise(async (resolve) => {

        if (!browser) {
          resolve(null)
          return
        }

        // console.log("Starting %s", key)
        const page = await browser.newPage()
        await page.setContent(html)
        const pdf = await page.pdf({
          format: 'A4',
          printBackground: true,
          // displayHeaderFooter: true,
          // margin: { top: '1.8cm', right: '1cm', bottom: '1cm', left: '1cm' },
        })

        // await page.setContent(html, {
        //   waitUntil: ['domcontentloaded', 'networkidle0', 'load'],
        // })
        //
        // await page.evaluate('window.scrollTo(0, document.body.scrollHeight)')
        //
        // const result = await page.pdf({format: 'a4', printBackground: true})

        // console.log("PDF %s", key)
        const command = new PutObjectCommand({
          Key: key,
          Bucket: Resource.ReceiptsBucket.name,
          Body: pdf,
        });

        await s3.send(command);
        // console.log("UPLOADED %s", key)
        resolve(key)
      })

      promises.push(promise)
    }

    await Promise.all(promises)

  } catch (e) {
    console.log('Chromium error', {e})
  } finally {
    if (browser !== null) {
      await browser.close()
    }
  }

  // console.log("SUCCESS")
  return {
    statusCode: 200,
    body: "OK",
  }
}

function decodeBase64UrlStr(encoded: string) {

  let base64 = encoded.replace(/-/g, '+').replace(/_/g, '/');

  const padding = base64.length % 4;
  if (padding) {
    base64 += '='.repeat(4 - padding);
  }

  const binaryString = atob(base64);

  const byteNumbers = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    byteNumbers[i] = binaryString.charCodeAt(i);
  }

  const decoder = new TextDecoder('utf-8');
  return decoder.decode(byteNumbers);
}
