import { createContext, useContext, useEffect, useState, useCallback } from 'react';
import type { ReactNode } from 'react';
import { useAuth } from './AuthContext';

interface WebSocketContextType {
  isConnected: boolean;
  subscribeToKV: (callback: (data: KVEvent) => void) => () => void;
  subscribeToServices: (callback: (data: ServiceEvent) => void) => () => void;
  subscribeToHealth: (callback: (data: HealthEvent) => void) => () => void;
}

interface KVEvent {
  type: 'set' | 'delete';
  key: string;
  value?: string;
  timestamp: string;
}

interface ServiceEvent {
  type: 'register' | 'deregister' | 'heartbeat' | 'expired';
  service: {
    id: string;
    name: string;
    address: string;
    port: number;
    status?: string;
  };
  timestamp: string;
}

interface HealthEvent {
  type: 'health_update';
  data: {
    status: string;
    uptime: number;
    services?: {
      total: number;
      healthy: number;
      unhealthy: number;
    };
  };
  timestamp: string;
}

const WebSocketContext = createContext<WebSocketContextType | undefined>(undefined);

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const [isConnected, setIsConnected] = useState(false);
  const [ws, setWs] = useState<WebSocket | null>(null);
  const [kvSubscribers, setKvSubscribers] = useState<Set<(data: KVEvent) => void>>(new Set());
  const [serviceSubscribers, setServiceSubscribers] = useState<Set<(data: ServiceEvent) => void>>(new Set());
  const [healthSubscribers, setHealthSubscribers] = useState<Set<(data: HealthEvent) => void>>(new Set());
  const { token, isAuthenticated } = useAuth();

  const connect = useCallback(() => {
    if (!isAuthenticated) return;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const wsUrl = `${protocol}//${host}/ws/updates`;

    console.log('Connecting to WebSocket:', wsUrl);

    const socket = new WebSocket(wsUrl);

    socket.onopen = () => {
      console.log('WebSocket connected');
      setIsConnected(true);

      // Send auth token if available
      if (token) {
        socket.send(JSON.stringify({
          type: 'auth',
          token,
        }));
      }

      // Subscribe to all event types
      socket.send(JSON.stringify({
        type: 'subscribe',
        channels: ['kv', 'services', 'health'],
      }));
    };

    socket.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        console.log('WebSocket message:', message);

        // Route message to appropriate subscribers
        switch (message.channel) {
          case 'kv':
            kvSubscribers.forEach(callback => callback(message.data));
            break;
          case 'services':
            serviceSubscribers.forEach(callback => callback(message.data));
            break;
          case 'health':
            healthSubscribers.forEach(callback => callback(message.data));
            break;
        }
      } catch (error) {
        console.error('Error parsing WebSocket message:', error);
      }
    };

    socket.onerror = (error) => {
      console.error('WebSocket error:', error);
      setIsConnected(false);
    };

    socket.onclose = () => {
      console.log('WebSocket disconnected');
      setIsConnected(false);

      // Attempt to reconnect after 5 seconds
      setTimeout(() => {
        if (isAuthenticated) {
          connect();
        }
      }, 5000);
    };

    setWs(socket);
  }, [isAuthenticated, token, kvSubscribers, serviceSubscribers, healthSubscribers]);

  useEffect(() => {
    if (isAuthenticated && !ws) {
      connect();
    }

    return () => {
      if (ws) {
        ws.close();
      }
    };
  }, [isAuthenticated, ws, connect]);

  const subscribeToKV = useCallback((callback: (data: KVEvent) => void) => {
    setKvSubscribers(prev => new Set(prev).add(callback));
    return () => {
      setKvSubscribers(prev => {
        const next = new Set(prev);
        next.delete(callback);
        return next;
      });
    };
  }, []);

  const subscribeToServices = useCallback((callback: (data: ServiceEvent) => void) => {
    setServiceSubscribers(prev => new Set(prev).add(callback));
    return () => {
      setServiceSubscribers(prev => {
        const next = new Set(prev);
        next.delete(callback);
        return next;
      });
    };
  }, []);

  const subscribeToHealth = useCallback((callback: (data: HealthEvent) => void) => {
    setHealthSubscribers(prev => new Set(prev).add(callback));
    return () => {
      setHealthSubscribers(prev => {
        const next = new Set(prev);
        next.delete(callback);
        return next;
      });
    };
  }, []);

  return (
    <WebSocketContext.Provider
      value={{
        isConnected,
        subscribeToKV,
        subscribeToServices,
        subscribeToHealth,
      }}
    >
      {children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocket() {
  const context = useContext(WebSocketContext);
  if (context === undefined) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
}