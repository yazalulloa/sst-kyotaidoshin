package com.kyotaidoshin;

import com.amazonaws.services.lambda.runtime.Context;
import com.amazonaws.services.lambda.runtime.RequestStreamHandler;
import com.fasterxml.jackson.databind.ObjectReader;
import com.openhtmltopdf.pdfboxout.PdfBoxRenderer;
import com.openhtmltopdf.pdfboxout.PdfRendererBuilder;
import io.reactivex.rxjava3.core.Single;
import io.vertx.core.json.Json;
import io.vertx.rxjava3.core.buffer.Buffer;
import io.vertx.rxjava3.ext.web.client.WebClient;
import jakarta.inject.Named;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Base64;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

@Named("html_to_pdf")
public class HtmlToPdfLambda implements RequestStreamHandler {

  private static final Logger logger = LoggerFactory.getLogger(HtmlToPdfLambda.class);

  private final ObjectReader reader = Util.getObjectReader();
  private final WebClient webClient = Util.getWebClient();
  private final PdfRendererBuilder builder = Util.getPdfRendererBuilder();

  @Override
  public void handleRequest(InputStream inputStream, OutputStream outputStream, Context context) throws IOException {

    final var singles = new ArrayList<Single<String>>();

    try (final var iterator = reader.<HtmlToPdfRequest>readValues(inputStream)) {
      final var list = iterator.readAll();

      logger.info("Pdf requests {}", list.size());
      for (var pdfRequest : list) {

        if (pdfRequest.presignedUrl() == null) {
          logger.warn("No presigned url for {}", pdfRequest.objectKey());
          continue;
        }

        byte[] decodedBytes = Base64.getUrlDecoder().decode(pdfRequest.html());
        final var html = new String(decodedBytes);

        builder.withHtmlContent(html, "");

        try (final var byteArrayOutputStream = new ByteArrayOutputStream()) {
          builder.toStream(byteArrayOutputStream);
          try (PdfBoxRenderer renderer = builder.buildPdfRenderer()) {
            renderer.createPDF();
          } catch (IOException e) {
            throw new RuntimeException(e);
          }

          byteArrayOutputStream.close();

          logger.info("PDF size {}", byteArrayOutputStream.size());

          singles.add(webClient.putAbs(pdfRequest.presignedUrl())
              .rxSendBuffer(Buffer.buffer(byteArrayOutputStream.toByteArray()))
              .map(response -> {

                if (response.statusCode() != 200 && response.statusCode() != 204) {
                  throw new RuntimeException("Error uploading %s".formatted(response.statusCode()));
                }

                return pdfRequest.objectKey();
              }));

        }
      }
    }

    logger.info("Before singles {}", singles.size());
    final var res = Single.merge(singles)
        .toList()
        .map(Json::encode)
        .blockingGet();

    outputStream.write(res.getBytes(StandardCharsets.UTF_8));
    outputStream.close();


  }
}
