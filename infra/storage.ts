export const webAssetsBucket = new sst.aws.Bucket("WebAssetsBucket", {
  access: "cloudfront",
});

export const receiptsBucket = new sst.aws.Bucket("ReceiptsBucket", {
  versioning: false,
});