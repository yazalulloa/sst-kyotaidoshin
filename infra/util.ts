export const PROD_STAGE: string = "production";
export const DEV_STAGE: string = "dev";

export const isLocal = Boolean(
    $app.stage !== PROD_STAGE && $app.stage !== DEV_STAGE,
);
console.log("isLocal", isLocal);

export const isrPrefix = "isr/v6"

