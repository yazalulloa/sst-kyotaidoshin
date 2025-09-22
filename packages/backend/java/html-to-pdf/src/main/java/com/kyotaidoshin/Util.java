package com.kyotaidoshin;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.MapperFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.openhtmltopdf.pdfboxout.PdfRendererBuilder;
import io.reactivex.rxjava3.plugins.RxJavaPlugins;
import io.vertx.rxjava3.core.RxHelper;
import io.vertx.rxjava3.core.Vertx;

public class Util {

  private static PdfRendererBuilder pdfRendererBuilder;

  public static PdfRendererBuilder getPdfRendererBuilder() {
    if (pdfRendererBuilder != null) {
      return pdfRendererBuilder;
    }

    synchronized (Util.class) {
      if (pdfRendererBuilder != null) {
        return pdfRendererBuilder;
      }
      pdfRendererBuilder = new PdfRendererBuilder();
      pdfRendererBuilder.withProducer("kyotaidoshin");
    }
    return pdfRendererBuilder;
  }


  public static void configureJsonMappers(ObjectMapper objectMapper) {
    objectMapper.findAndRegisterModules();

    objectMapper
        .disable(SerializationFeature.FAIL_ON_EMPTY_BEANS)
        .disable(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES)
        .enable(MapperFeature.ACCEPT_CASE_INSENSITIVE_ENUMS)
        .setSerializationInclusion(JsonInclude.Include.NON_NULL)
    ;
  }

  public static void configureSchedulers(Vertx vertx) {
    RxJavaPlugins.setComputationSchedulerHandler(s -> RxHelper.scheduler(vertx));
    RxJavaPlugins.setIoSchedulerHandler(s -> RxHelper.blockingScheduler(vertx));
    RxJavaPlugins.setNewThreadSchedulerHandler(s -> RxHelper.scheduler(vertx));
  }


}
