package com.kyotaidoshin;

import com.amazonaws.services.lambda.runtime.Context;
import com.amazonaws.services.lambda.runtime.RequestStreamHandler;
import com.openhtmltopdf.pdfboxout.PdfBoxRenderer;
import io.reactivex.rxjava3.core.Single;
import io.vertx.core.json.Json;
import io.vertx.rxjava3.core.buffer.Buffer;
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


  @Override
  public void handleRequest(InputStream inputStream, OutputStream outputStream, Context context) throws IOException {

    final var builder = Util.getPdfRendererBuilder();
    final var reader = Util.getObjectReader();
    final var webClient = Util.getWebClient();

    builder.withProducer("kyotaidoshin");
    final var singles = new ArrayList<Single<String>>();

    try (final var iterator = reader.<HtmlToPdfRequest>readValues(inputStream)) {
      final var list = iterator.readAll();

      logger.info("Pdf requests {}", list.size());
      for (var pdfRequest : list) {
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

          if (pdfRequest.presignedUrl() != null) {
            singles.add(webClient.putAbs(pdfRequest.presignedUrl())
                .rxSendBuffer(Buffer.buffer(byteArrayOutputStream.toByteArray()))
                .map(response -> {

                  if (response.statusCode() != 200 && response.statusCode() != 204) {
                    throw new RuntimeException("Error uploading %s".formatted(response.statusCode()));
                  }

                  return pdfRequest.objectKey();
                }));
          } else {
            logger.info("No presigned url for {}", pdfRequest.objectKey());
          }

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
