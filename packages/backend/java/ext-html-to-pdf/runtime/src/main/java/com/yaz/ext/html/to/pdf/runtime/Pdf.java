package com.yaz.ext.html.to.pdf.runtime;

import com.openhtmltopdf.util.Diagnostic;
import com.openhtmltopdf.util.XRLog;
import com.openhtmltopdf.util.XRLogger;
import java.util.logging.Level;
import org.jboss.logging.Logger;

public class Pdf {

  private final static Logger logger = Logger.getLogger(Pdf.class);

  /*
   * For some reason, this is not automatically caught by JBoss Logging, but we also want to downgrade its info
   * level which is too verbose for us.
   */
  static {
    XRLog.setLoggerImpl(new XRLogger() {
      @Override
      public void log(String where, Level level, String msg) {
        // FIXME: dropped where because if I set it, it appears as <unknown>" in the logs and I can't even enable the logs
        logger.log(translate(level), msg);
      }

      @Override
      public void log(String where, Level level, String msg, Throwable th) {
        // FIXME: dropped where because if I set it, it appears as <unknown>" in the logs and I can't even enable the logs
        logger.log(translate(level), msg, th);
      }

      private org.jboss.logging.Logger.Level translate(Level level) {
        // downgrade INFO which is too verbose
        if (level == Level.INFO) {
          return Logger.Level.DEBUG;
        }
        // the rest is a best guess
        if (level == Level.WARNING) {
          return Logger.Level.WARN;
        }
        if (level == Level.SEVERE) {
          return Logger.Level.ERROR;
        }
        if (level == Level.FINE
            || level == Level.FINER) {
          return Logger.Level.TRACE;
        }
        if (level == Level.FINEST) {
          return Logger.Level.DEBUG;
        }
        return Logger.Level.INFO;
      }

      @Override
      public void setLevel(String logger, Level level) {
      }

      @Override
      public boolean isLogLevelEnabled(Diagnostic diagnostic) {
        return logger.isEnabled(translate(diagnostic.getLevel()));
      }
    });
  }
}
