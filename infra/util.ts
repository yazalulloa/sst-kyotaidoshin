export const PROD_STAGE: string = "production";
export const STAG_STAGE: string = "staging";

export const isLocal = Boolean(
    $app.stage !== PROD_STAGE && $app.stage !== STAG_STAGE,
);
console.log("isLocal", isLocal);

export const isrPrefix = "isr/v6"