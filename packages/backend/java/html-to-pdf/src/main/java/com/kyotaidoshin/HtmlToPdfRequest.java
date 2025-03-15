package com.kyotaidoshin;

public record HtmlToPdfRequest(
    String objectKey,
    String html,
    String presignedUrl
) {

}
