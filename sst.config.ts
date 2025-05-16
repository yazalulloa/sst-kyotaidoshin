/// <reference path="./.sst/platform/config.d.ts" />


export default $config({
  app(input) {
    return {
      name: "kyotaidoshin",
      removal: input?.stage === "production" ? "retain" : "remove",
      protect: ["production"].includes(input?.stage),
      home: "aws",
      providers: {aws: "6.71.0"},
    };
  },
  async run() {

    const infra = await import("./infra");


    return {
      SiteUrl: infra.site.url,
    };
  },
});
