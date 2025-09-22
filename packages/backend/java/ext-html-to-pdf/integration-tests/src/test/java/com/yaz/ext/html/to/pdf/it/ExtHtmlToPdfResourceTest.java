package com.yaz.ext.html.to.pdf.it;

import static io.restassured.RestAssured.given;
import static org.hamcrest.Matchers.is;

import org.junit.jupiter.api.Test;

import io.quarkus.test.junit.QuarkusTest;

@QuarkusTest
public class ExtHtmlToPdfResourceTest {

    @Test
    public void testHelloEndpoint() {
        given()
                .when().get("/ext-html-to-pdf")
                .then()
                .statusCode(200)
                .body(is("Hello ext-html-to-pdf"));
    }
}
