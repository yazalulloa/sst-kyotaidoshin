package com.kyotaidoshin;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.MapperFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.ObjectReader;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.openhtmltopdf.pdfboxout.PdfRendererBuilder;
import io.reactivex.rxjava3.plugins.RxJavaPlugins;
import io.vertx.core.json.jackson.DatabindCodec;
import io.vertx.rxjava3.core.RxHelper;
import io.vertx.rxjava3.core.Vertx;
import io.vertx.rxjava3.ext.web.client.WebClient;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class Util {

  private static final Logger logger = LoggerFactory.getLogger(Util.class);


  private static ObjectReader objectReader;

  public static ObjectReader getObjectReader() {

    if (objectReader != null) {
      return objectReader;
    }

    synchronized (Util.class) {
      if (objectReader != null) {
        return objectReader;
      }

      configureJsonMappers(DatabindCodec.mapper());
      objectReader = DatabindCodec.mapper().readerFor(HtmlToPdfRequest.class);
    }
    return objectReader;
  }

  private static Vertx vertx;

  public static Vertx getVertx() {
    if (vertx != null) {
      return vertx;
    }

    synchronized (Util.class) {
      if (vertx != null) {
        return vertx;
      }
      vertx = Vertx.vertx();
      configureSchedulers(vertx);
    }
    return vertx;
  }

  private static WebClient webClient;

  public static WebClient getWebClient() {
    if (webClient != null) {
      return webClient;
    }

    synchronized (Util.class) {
      if (webClient != null) {
        return webClient;
      }
      webClient = WebClient.create(getVertx());
    }
    return webClient;
  }

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
