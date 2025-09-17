package ru.nsu.fit.selffinder;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;

import java.io.IOException;
import java.net.*;
import java.time.LocalDateTime;
import java.util.Random;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public class MulticastPeer implements AutoCloseable {
    private static final Logger log = LogManager.getLogger(MulticastPeer.class);
    private static final String MSG = "I am copy";
    private MulticastSocket socket;
    private final InetAddress group;
    private final Integer port;
    private final ConcurrentHashMap<String, LocalDateTime> peers = new ConcurrentHashMap<>();
    private final ExecutorService executor = Executors.newCachedThreadPool();
    private final long appId = new Random().nextLong();

    @Override
    public void close() {
        executor.shutdownNow();
        byte[] buf = (MSG + appId + "BYE").getBytes();
        DatagramPacket packet = new DatagramPacket(buf, buf.length, group, port);
        try {
            socket.send(packet);
        } catch (IOException e) {
            log.error(e);
        }
        socket.close();
    }

    public MulticastPeer(InetAddress group, Integer port) {
        this.group = group;
        this.port = port;
        try {
            socket = new MulticastSocket(port);
            socket.joinGroup(new InetSocketAddress(group, port), NetworkInterface.getByName("wlo1"));
        } catch (IOException e) {
            log.error(e);
        }
        initThreads();
    }

    public void initThreads() {
        startSender();
        startReceiver();
    }

    private void startReceiver() {
        CompletableFuture.runAsync(() -> {
            try {
                while (!Thread.currentThread().isInterrupted()) {
                    byte[] buf = new byte[4096];
                    DatagramPacket packet = new DatagramPacket(buf, buf.length);
                    socket.receive(packet);
                    String received = new String(packet.getData(), 0, packet.getLength());
                    if (!received.startsWith(MSG)) {
                        continue;
                    }
                    String key = packet.getAddress().getHostAddress() + ":" + packet.getPort() + " " + received.substring(MSG.length()).replace("BYE", "");
                    if (received.contains("BYE")) {
                        peers.remove(key);
                        System.out.println(peers);
                    } else if (!peers.containsKey(key)) {
                        peers.put(key, LocalDateTime.now());
                        System.out.println(peers);
                    } else {
                        peers.put(key, LocalDateTime.now());
                    }
                    packet.setLength(buf.length);
                }

            } catch (IOException e) {
                log.error(e);
                Thread.currentThread().interrupt();
            }
        }, executor);
    }

    private void startSender() {
        CompletableFuture.runAsync(() -> {
            byte[] buf = (MSG + appId).getBytes();
            DatagramPacket packet = new DatagramPacket(buf, buf.length, group, port);
            try {
                while (!Thread.currentThread().isInterrupted()) {
                    socket.send(packet);
                    Thread.sleep(3000);
                }
            } catch (IOException | InterruptedException e) {
                log.error(e);
                Thread.currentThread().interrupt();
            }
        }, executor);
    }
}
