package ru.nsu.fit.tcptransferring.server;

import jakarta.annotation.PostConstruct;
import lombok.RequiredArgsConstructor;
import org.springframework.scheduling.annotation.Async;
import org.springframework.stereotype.Service;

import java.io.IOException;
import java.net.Socket;
import java.util.concurrent.Executor;
import java.util.concurrent.Executors;


@Service
@RequiredArgsConstructor
public class ClientHandler {
    private final Server server;
    private final Executor executor = Executors.newCachedThreadPool();

    @PostConstruct
    public void init() {
        try {
            run();
        }catch (IOException e) {
            Thread.currentThread().interrupt();
        }
    }

    @Async
    public void run() throws IOException {
            while (!Thread.currentThread().isInterrupted()) {
                Socket clientSocket = server.getServerSocket().accept();
                Loader loader = new Loader(clientSocket);
                executor.execute(loader);
            }

    }

}
