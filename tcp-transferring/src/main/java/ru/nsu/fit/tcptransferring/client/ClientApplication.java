package ru.nsu.fit.tcptransferring.client;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableAsync;

@EnableAsync
@SpringBootApplication(scanBasePackages = "ru.nsu.fit.tcptransferring.client")
public class ClientApplication {
    public static void main(String[] args) {
        SpringApplication.run(ru.nsu.fit.tcptransferring.client.ClientApplication.class, args);
    }

}