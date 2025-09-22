package com.kyotaidoshin;

import com.openhtmltopdf.util.XRLog;
import io.quarkus.runtime.StartupEvent;
import io.vertx.core.json.jackson.DatabindCodec;
import io.vertx.rxjava3.core.Vertx;
import jakarta.enterprise.context.ApplicationScoped;
import jakarta.enterprise.event.Observes;
import lombok.RequiredArgsConstructor;

@RequiredArgsConstructor
@ApplicationScoped
public class Startup {

  private final Vertx vertx;

  /**
   * This method is executed at the start of your application
   */
  public void start(@Observes StartupEvent evt) {
    Util.configureSchedulers(vertx);
    Util.configureJsonMappers(DatabindCodec.mapper());

    XRLog.listRegisteredLoggers().forEach(logger -> XRLog.setLevel(logger, java.util.logging.Level.OFF));
  }
}
