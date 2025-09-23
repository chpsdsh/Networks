package ru.nsu.fit.tcptransferring.server;

import lombok.RequiredArgsConstructor;
import org.springframework.scheduling.annotation.Async;
import org.springframework.stereotype.Service;

import java.net.Socket;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;


@Service
@RequiredArgsConstructor
public class ClientHandler {
    private final Server server;

    @Async
    public void run() {
        ExecutorService executor = Executors.newSingleThreadExecutor();
        executor.execute(() -> {
            while (!Thread.currentThread().isInterrupted()) {
                Socket clientSocket = server.getServerSocket().accept();

            }

        });
    }

}
