package ru.nsu.fit.selffinder;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;

import java.net.InetAddress;
import java.net.UnknownHostException;
import java.util.concurrent.CountDownLatch;


public class Main {
    private static final Logger log = LogManager.getLogger(Main.class);

    public static void main(String[] args) {
        String address = args[0];
        Integer port = Integer.parseInt(args[1]);
        try {
            InetAddress group = InetAddress.getByName(address);
            CountDownLatch stop = new CountDownLatch(1);
            Runtime.getRuntime().addShutdownHook(new Thread(stop::countDown));
            try (MulticastPeer peer = new MulticastPeer(group, port)) {
                stop.await();
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        } catch (UnknownHostException e) {
            log.error(e);
            Thread.currentThread().interrupt();
        }
    }
}