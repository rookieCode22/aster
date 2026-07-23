import { useCallback, useEffect, useRef, useState } from 'react';
import { getToken } from '../api/client';

interface Message {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: string;
}

export function useWebSocket(sessionId: string | null) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectRef = useRef<number>(0);

  const connect = useCallback(() => {
    if (!sessionId) return;
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(
      `${proto}//${location.host}/ws/chat/${sessionId}?token=${getToken()}`,
    );
    wsRef.current = ws;

    ws.onopen = () => {
      setConnected(true);
      reconnectRef.current = 0;
    };

    ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data) as Message;
        setMessages((prev) => [...prev, msg]);
      } catch {
        // ignore parse errors
      }
    };

    ws.onclose = () => {
      setConnected(false);
      // exponential backoff reconnect, max 3 attempts
      if (reconnectRef.current < 3) {
        reconnectRef.current++;
        setTimeout(connect, 1000 * reconnectRef.current);
      }
    };
  }, [sessionId]);

  useEffect(() => {
    connect();
    return () => wsRef.current?.close();
  }, [connect]);

  const send = useCallback((content: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    wsRef.current.send(JSON.stringify({ type: 'message', content }));
  }, []);

  const clearMessages = useCallback(() => setMessages([]), []);

  return { messages, connected, send, clearMessages };
}
