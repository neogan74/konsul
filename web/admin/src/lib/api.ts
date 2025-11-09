import axios from 'axios';

const api = axios.create({
  baseURL: 'http://localhost:8888',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Types
export interface Service {
  id: string;
  name: string;
  address: string;
  port: number;
  tags: string[];
  metadata?: Record<string, string>;
  check?: {
    ttl?: string;
    http?: string;
    interval?: string;
  };
  status?: string;
  last_heartbeat?: string;
}

export interface KVPair {
  key: string;
  value: string;
  flags?: number;
  create_index?: number;
  modify_index?: number;
}

export interface HealthResponse {
  status: string;
  uptime: number;
  timestamp: string;
  version?: string;
  services?: {
    total: number;
    healthy: number;
    unhealthy: number;
  };
  kv?: {
    total_keys: number;
  };
  system?: {
    goroutines: number;
    memory: {
      alloc: number;
      total_alloc: number;
      sys: number;
      heap_alloc: number;
      heap_sys: number;
    };
  };
}

export interface RegisterServiceRequest {
  id?: string;
  name: string;
  address: string;
  port: number;
  tags?: string[];
  metadata?: Record<string, string>;
  check?: {
    ttl?: string;
    http?: string;
    interval?: string;
  };
}

// Service API
export const getServices = async (): Promise<Service[]> => {
  try {
    const response = await api.get('/services/');
    return response.data || [];
  } catch (error) {
    console.error('Error fetching services:', error);
    return [];
  }
};

export const registerService = async (service: RegisterServiceRequest): Promise<void> => {
  await api.put('/register', service);
};

export const deregisterService = async (serviceName: string): Promise<void> => {
  await api.delete(`/deregister/${serviceName}`);
};

export const sendHeartbeat = async (serviceName: string): Promise<void> => {
  await api.put(`/heartbeat/${serviceName}`);
};

export const getService = async (serviceName: string): Promise<Service[]> => {
  const response = await api.get(`/services/${serviceName}`);
  return response.data || [];
};

// KV Store API
export const getKV = async (key: string): Promise<KVPair | null> => {
  try {
    const response = await api.get(`/kv/${key}`);
    return { key, value: response.data };
  } catch (error) {
    console.error('Error fetching KV:', error);
    return null;
  }
};

export const setKV = async (key: string, value: string): Promise<void> => {
  await api.put(`/kv/${key}`, { value });
};

export const deleteKV = async (key: string): Promise<void> => {
  await api.delete(`/kv/${key}`);
};

export const listKV = async (prefix: string = ''): Promise<KVPair[]> => {
  try {
    const response = await api.get('/kv/');
    if (response.data && Array.isArray(response.data)) {
      return response.data.map((key: string) => ({
        key,
        value: '',
      }));
    }
    return [];
  } catch (error) {
    console.error('Error listing KV:', error);
    return [];
  }
};

export const listKVWithValues = async (prefix: string = ''): Promise<KVPair[]> => {
  try {
    const response = await api.get('/kv/');
    if (!response.data || !Array.isArray(response.data)) {
      return [];
    }

    // Fetch values for all keys in parallel
    const keys = response.data;
    const kvPairs = await Promise.all(
      keys.map(async (key: string) => {
        try {
          const valueResponse = await api.get(`/kv/${key}`);
          return {
            key,
            value: valueResponse.data,
          };
        } catch {
          return { key, value: '' };
        }
      })
    );

    return kvPairs;
  } catch (error) {
    console.error('Error listing KV with values:', error);
    return [];
  }
};

// Health API
export const getHealth = async (): Promise<HealthResponse> => {
  const response = await api.get('/health');
  return response.data;
};

export default api;
