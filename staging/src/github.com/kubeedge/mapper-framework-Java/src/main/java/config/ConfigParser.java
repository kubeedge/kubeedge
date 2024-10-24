package config;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import lombok.extern.slf4j.Slf4j;
import model.Config;

import java.io.File;
import java.io.IOException;
@Slf4j
public class ConfigParser {
    private static final String defaultConfigFile = "src/main/resources/config.yaml";
    public static Config parse() {
        ObjectMapper mapper = new ObjectMapper(new YAMLFactory());

        try {
            // Reading YAML file
            return mapper.readValue(new File(defaultConfigFile), Config.class);
        } catch (IOException e) {
            log.error("Fail to read file: {} with err: {}", defaultConfigFile, e.getMessage());
        }
        return null;
    }
}
