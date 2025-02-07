import type {Handler} from "aws-lambda";
import {GetObjectCommand, NoSuchKey, S3Client, S3ServiceException,} from "@aws-sdk/client-s3";
import type {Range, WorkSheet} from 'xlsx';
import * as XLSX from 'xlsx';
import {SendMessageCommand, SQSClient} from "@aws-sdk/client-sqs";
import {Resource} from "sst";
import {v4 as uuidv4} from 'uuid';

export const handler: Handler = async (event, context) => {

  console.log("Event: ", event);

  // let body = event.body
  // if (!body) {
  //   return {
  //     statusCode: 400,
  //     body: "Body is required",
  //   }
  // }
  //
  // console.log("Body: ", body);
  //
  // let json = JSON.parse(body);

  let bucket = event?.bucket?.trim()
  if (!bucket || bucket.length == 0) {
    return {
      statusCode: 400,
      body: "Bucket is required",
    }
  }

  let key = event?.key?.trim()
  if (!key || key.length == 0) {
    return {
      statusCode: 400,
      body: "Key is required",
    }
  }

  const client = new S3Client({});

  try {
    const response = await client.send(
        new GetObjectCommand({
          Bucket: bucket,
          Key: key,
        }),
    );
    // The Body object also has 'transformToByteArray' and 'transformToWebStream' methods.
    let body = response.Body;
    if (!body) {
      return {
        statusCode: 500,
        body: "No body in response",
      }
    }


    let byteArray = await body.transformToByteArray();
    let workBook = XLSX.read(byteArray, {type: 'array'});

    let sqsClient = new SQSClient()

    for (let sheetIndex = 0; sheetIndex < workBook.SheetNames.length; sheetIndex++) {
      let sheetName = workBook.SheetNames[sheetIndex]

      console.log(`Sheet Name: ${sheetName}`);
      const worksheet = workBook.Sheets[sheetName];
      let ref = worksheet['!ref']
      if (!ref) {
        return;
      }

      const range = XLSX.utils.decode_range(ref);
      let dateOfFile: string | undefined = undefined;
      let altDateOfFile: string | undefined = undefined;
      let dateOfRate: Date | undefined = undefined;

      interface Rate {
        id: number,
        from_currency: string,
        rate: number,
        date_of_rate: Date,
        date_of_file: string | undefined,
        alt_date_of_file: string | undefined,
      }

      let rateArray: Rate[] = [];
      for (let row = range.s.r; row <= range.e.r; row++) {

        // Iterate through each column in the row

        if (row == 0) {

          let rowData = parseRow(worksheet, row, range);
          let cellDate = dateCell(rowData);

          if (cellDate) {
            let split = cellDate.split(" ");
            let dateSplit = split[0].split("/")
            let day = parseInt(dateSplit[0])
            let month = parseInt(dateSplit[1])
            let year = parseInt(dateSplit[2])

            let timeSplit = split[1].split(":")
            let hour = parseInt(timeSplit[0])
            let minute = parseInt(timeSplit[1])


            let date = new Date(year, month - 1, day, hour, minute, 0);
            // console.log("Date: ", date);
            dateOfFile = cellDate
          }
        }

        if (row == 4) {
          let rowData = parseRow(worksheet, row, range);

          if (!dateOfFile) {
            let cellDate: string = rowData[1]
            altDateOfFile = cellDate.substring(cellDate.indexOf(":") + 1).trim()
             console.log(`Alternative Date: ${altDateOfFile}`);
          }

          let cellValue: string = rowData[3]
          if (cellValue.length === 0) {
            cellValue = rowData[2]
          }
          let split = cellValue.substring(cellValue.indexOf(":") + 1).trim().split("/")
          let day = parseInt(split[0])
          let month = parseInt(split[1])
          let year = parseInt(split[2])

          console.log("Cell Value: ", cellValue);
          dateOfRate = new Date(year, month, day, 0, 0, 0);
          console.log(`Date of Rate: ${dateOfRate}`);
        }

        if (row > 9 && dateOfRate !== undefined) {
          let rowData = parseRow(worksheet, row, range);
          let currency: string = rowData[0]

          if (currency.length != 3) {
            currency = rowData[1]
            if (currency.length != 3) {
              break
            }
          }

          let rate: number = rowData[6]
          // console.log(`Currency: ${currency} Rate: ${rate}`);
          // console.log("Type of cell6: ", typeof cell6);
          // let rate = parseFloat(cell6.replace(",", ""))

          // console.log("Date of Rate: ", dateOfRate);
          let str = `${dateOfRate.getFullYear()}${dateOfRate.getMonth().toString().padStart(2, '0')}${dateOfRate.getDay().toString().padStart(2, '0')}${toASCII(currency)}${sheetIndex.toString().padStart(4, '0')}`
          // console.log(`ID: ${str}`);
          let id = parseInt(str)

          rateArray.push({
            id: id,
            from_currency: currency,
            rate: rate,
            date_of_rate: dateOfRate,
            date_of_file: dateOfFile,
            alt_date_of_file: altDateOfFile,
          })

        }


      }

      console.log(`Rates: ${rateArray.length}`);

      if (rateArray.length > 0 && false) {
        let json: string = JSON.stringify({
          rates: rateArray,
        })

        // console.log("JSON: ", json);

        await sqsClient.send(new SendMessageCommand({
          QueueUrl: Resource.ProcessRatesQueue.url,
          MessageBody: json,
          MessageGroupId: "processing-rates",
          MessageDeduplicationId: uuidv4(),
        }))
      }
    }

    return workBook.SheetNames.length
  } catch (caught) {
    if (caught instanceof NoSuchKey) {
      console.error(
          `Error from S3 while getting object "${key}" from "${bucket}". No such key exists.`,
      );
    } else if (caught instanceof S3ServiceException) {
      console.error(
          `Error from S3 while getting object from ${bucket} ${key}.  ${caught.name}: ${caught.message}`,
      );
    } else {
      throw caught;
    }
  }
  return context.logStreamName;
}


function toASCII(str: string): string {
  let array: string[] = []
  for (let i = 0; i < str.length; i++) {
    array.push(str.charCodeAt(i).toString())
  }
  return array.join("")
}

function dateCell(rowData: any[]): string | undefined {
  let cell5: string = rowData[5]

  if (cell5.length > 0) {
    return cell5;
  }

  let cell6: string = rowData[6]
  if (cell6.length > 0) {
    return cell6;
  }
  console.log("Row: ", rowData);
  return undefined;
}

function parseRow(worksheet: WorkSheet, row: number, range: Range): any[] {
  const rowData = [];
  for (let col = range.s.c; col <= range.e.c; col++) {
    const cellAddress = XLSX.utils.encode_cell({r: row, c: col});
    const cell = worksheet[cellAddress]; // Get the cell

    // Push the cell value or an empty string if the cell is undefined
    rowData.push(cell ? cell.v : '');
  }
  return rowData;
}