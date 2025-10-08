package ru.nsu.fit.tcptransferring.server;

import jakarta.annotation.PostConstruct;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

import java.io.IOException;
import java.net.Socket;
import java.util.concurrent.Executor;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;


@Service
@RequiredArgsConstructor
public class ClientHandler {
    private final Server server;
    private final Executor executor = Executors.newCachedThreadPool();
    private final ScheduledExecutorService scheduledExecutor = Executors.newSingleThreadScheduledExecutor();

    @PostConstruct
    public void init() {
        try {
            run();
        } catch (IOException e) {
            Thread.currentThread().interrupt();
        }
    }

    public void run() throws IOException {
        while (!Thread.currentThread().isInterrupted()) {
            Socket clientSocket = server.getServerSocket().accept();
            executor.execute(new Loader(clientSocket, scheduledExecutor));
        }
    }
}
