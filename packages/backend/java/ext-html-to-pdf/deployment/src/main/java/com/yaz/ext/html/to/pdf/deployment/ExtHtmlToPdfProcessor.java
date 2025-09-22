package com.yaz.ext.html.to.pdf.deployment;

import io.quarkus.bootstrap.classloading.QuarkusClassLoader;
import io.quarkus.deployment.annotations.BuildProducer;
import io.quarkus.deployment.annotations.BuildStep;
import io.quarkus.deployment.builditem.FeatureBuildItem;
import io.quarkus.deployment.builditem.nativeimage.NativeImageResourceBuildItem;
import io.quarkus.deployment.builditem.nativeimage.NativeImageResourceDirectoryBuildItem;
import io.quarkus.deployment.builditem.nativeimage.RuntimeInitializedClassBuildItem;
import java.util.List;
import org.jboss.logging.Logger;

class ExtHtmlToPdfProcessor {

  private static final Logger logger = Logger.getLogger(ExtHtmlToPdfProcessor.class);

  private static final String FEATURE = "ext-html-to-pdf";
  private static final String PDFBOX_PROBLEMATIC_CLASS = "org.apache.pdfbox.pdmodel.encryption.PublicKeySecurityHandler";
  private static final String PDF_RESPONSE_HANDLER_CLASS = "com.yaz.ext.html.to.pdf.runtime.Pdf";
//  private static final String PDF_RESPONSE_HANDLER_CLASS = "io.quarkiverse.renarde.pdf.runtime.PdfResponseHandler";

  @BuildStep
  FeatureBuildItem feature() {
    return new FeatureBuildItem(FEATURE);
  }

  @BuildStep
  void setupPdfBox(BuildProducer<RuntimeInitializedClassBuildItem> runtimeInitializedClassBuildItem,
      BuildProducer<NativeImageResourceBuildItem> nativeImageResourceBuildItem,
      BuildProducer<NativeImageResourceDirectoryBuildItem> resource) {

    logger.info("Setting up HTML to PDF extension");

    // If we have the renarde-pdf module, we'll see this class
    if (QuarkusClassLoader.isClassPresentAtRuntime(PDF_RESPONSE_HANDLER_CLASS)) {
      logger.info("PDF response handler class found");
      // Perhaps try to unify with https://github.com/quarkiverse/quarkus-pdfbox ?

      // This one needs to be initialised at runtime on jdk21/graalvm 23.1 because setting the logger starts the java2d disposer thread
      runtimeInitializedClassBuildItem.produce(new RuntimeInitializedClassBuildItem(PDF_RESPONSE_HANDLER_CLASS));
      // This one starts some crypto stuff
      runtimeInitializedClassBuildItem.produce(new RuntimeInitializedClassBuildItem(PDFBOX_PROBLEMATIC_CLASS));
      // This is started by anybody doing graphics at startup time, including pdfbox instantiating an empty image
      runtimeInitializedClassBuildItem.produce(new RuntimeInitializedClassBuildItem("sun.java2d.Disposer"));
      // This causes the pdfbox to log at static init time, which creates a JUL which is forbidden
      runtimeInitializedClassBuildItem
          .produce(new RuntimeInitializedClassBuildItem("com.openhtmltopdf.resource.FSEntityResolver"));
      // These call java/awt stuff at static init, which may initialise Java2D
      runtimeInitializedClassBuildItem
          .produce(new RuntimeInitializedClassBuildItem("com.openhtmltopdf.java2d.image.AWTFSImage"));
      runtimeInitializedClassBuildItem
          .produce(new RuntimeInitializedClassBuildItem("com.openhtmltopdf.java2d.image.AWTFSImage$NullImage"));
      runtimeInitializedClassBuildItem
          .produce(new RuntimeInitializedClassBuildItem("com.openhtmltopdf.pdfboxout.PdfBoxFastOutputDevice"));
      // These are needed at runtime for native image, and missing from quarkiverse-pdfbox
      nativeImageResourceBuildItem.produce(
          new NativeImageResourceBuildItem(List.of("resources/css/XhtmlNamespaceHandler.css",
              "resources/schema/openhtmltopdf/catalog-special.xml",
              "resources/schema/openhtmltopdf/char-entities-xhtml-only.ent",
              "resources/schema/openhtmltopdf/char-entities-xhtml-mathml.ent")));
      resource.produce(new NativeImageResourceDirectoryBuildItem("org/apache/pdfbox/resources/ttf"));

    }
  }
}
