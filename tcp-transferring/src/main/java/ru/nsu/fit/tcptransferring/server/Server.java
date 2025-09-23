package ru.nsu.fit.tcptransferring.server;

import lombok.Getter;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.net.ServerSocket;

@Getter
@Component
public class Server implements AutoCloseable {
    private final ServerSocket serverSocket;

    @Override
    public void close() throws IOException {
        serverSocket.close();
    }

    public Server(@Value("${server.port}") int port) throws IOException {
        serverSocket = new ServerSocket(port);
    }


}
