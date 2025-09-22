package com.kyotaidoshin;

import com.openhtmltopdf.pdfboxout.PdfRendererBuilder;
import io.vertx.rxjava3.core.Vertx;
import io.vertx.rxjava3.ext.web.client.WebClient;
import jakarta.inject.Singleton;
import jakarta.ws.rs.Produces;

public class Producers {

  @Singleton
  @Produces
  Vertx rxVertx(io.vertx.mutiny.core.Vertx vertx) {
    return Vertx.newInstance(vertx.getDelegate());
  }

  @Singleton
  @Produces
  WebClient webClient(Vertx vertx) {
    return WebClient.create(vertx);
  }

  @Singleton
  @Produces
  PdfRendererBuilder pdfRendererBuilder() {
    return Util.getPdfRendererBuilder();
  }
}
