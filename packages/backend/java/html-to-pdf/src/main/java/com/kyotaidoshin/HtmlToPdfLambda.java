package com.kyotaidoshin;

import com.amazonaws.services.lambda.runtime.Context;
import com.amazonaws.services.lambda.runtime.RequestHandler;
import com.openhtmltopdf.pdfboxout.PdfBoxRenderer;
import com.openhtmltopdf.pdfboxout.PdfRendererBuilder;
import io.reactivex.rxjava3.core.Single;
import io.vertx.core.json.Json;
import io.vertx.rxjava3.core.buffer.Buffer;
import io.vertx.rxjava3.ext.web.client.WebClient;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.util.ArrayList;
import java.util.Base64;
import java.util.List;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;

@Slf4j
@RequiredArgsConstructor
public class HtmlToPdfLambda implements RequestHandler<List<HtmlToPdfRequest>, String> {

  private final PdfRendererBuilder builder;
  private final WebClient webClient;

  @Override
  public String handleRequest(List<HtmlToPdfRequest> input, Context context) {

    final var singles = new ArrayList<Single<String>>();

    for (var pdfRequest : input) {

      if (pdfRequest.presignedUrl() == null) {
        log.warn("No presigned url for {}", pdfRequest.objectKey());
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

        log.info("PDF size {}", byteArrayOutputStream.size());

        singles.add(webClient.putAbs(pdfRequest.presignedUrl())
            .rxSendBuffer(Buffer.buffer(byteArrayOutputStream.toByteArray()))
            .map(response -> {

              if (response.statusCode() != 200 && response.statusCode() != 204) {
                throw new RuntimeException("Error uploading %s".formatted(response.statusCode()));
              }

              return pdfRequest.objectKey();
            }));

      } catch (IOException e) {
        throw new RuntimeException(e);
      }
    }

    log.info("Before singles {}", singles.size());
    return Single.merge(singles)
        .toList()
        .map(Json::encode)
        .blockingGet();
  }
}
