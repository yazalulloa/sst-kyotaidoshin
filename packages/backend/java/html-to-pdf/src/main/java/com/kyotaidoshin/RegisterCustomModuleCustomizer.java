package com.kyotaidoshin;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.quarkus.jackson.ObjectMapperCustomizer;
import jakarta.inject.Singleton;

@Singleton
public class RegisterCustomModuleCustomizer implements ObjectMapperCustomizer {

  public void customize(ObjectMapper mapper) {
    Util.configureJsonMappers(mapper);
  }
}
