package com.kyotaidoshin;

import io.quarkus.runtime.annotations.RegisterForReflection;

@RegisterForReflection
public record HtmlToPdfRequest(
    String objectKey,
    String html,
    String presignedUrl
) {

}
