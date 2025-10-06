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
    const response = await api.get('/v1/catalog/services');
    return response.data || [];
  } catch (error) {
    console.error('Error fetching services:', error);
    return [];
  }
};

export const registerService = async (service: RegisterServiceRequest): Promise<void> => {
  await api.put('/v1/agent/service/register', service);
};

export const deregisterService = async (serviceId: string): Promise<void> => {
  await api.put(`/v1/agent/service/deregister/${serviceId}`);
};

export const sendHeartbeat = async (checkId: string): Promise<void> => {
  await api.put(`/v1/agent/check/pass/${checkId}`);
};

export const getService = async (serviceName: string): Promise<Service[]> => {
  const response = await api.get(`/v1/catalog/service/${serviceName}`);
  return response.data || [];
};

// KV Store API
export const getKV = async (key: string): Promise<KVPair | null> => {
  try {
    const response = await api.get(`/v1/kv/${key}`);
    if (response.data && response.data.length > 0) {
      return response.data[0];
    }
    return null;
  } catch (error) {
    console.error('Error fetching KV:', error);
    return null;
  }
};

export const setKV = async (key: string, value: string, flags?: number): Promise<void> => {
  const params = flags !== undefined ? { flags } : {};
  await api.put(`/v1/kv/${key}`, value, { params });
};

export const deleteKV = async (key: string, recurse: boolean = false): Promise<void> => {
  const params = recurse ? { recurse: 'true' } : {};
  await api.delete(`/v1/kv/${key}`, { params });
};

export const listKV = async (prefix: string = ''): Promise<KVPair[]> => {
  try {
    const url = prefix ? `/v1/kv/${prefix}` : '/v1/kv/';
    const response = await api.get(url, { params: { keys: 'true' } });

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
    const url = prefix ? `/v1/kv/${prefix}` : '/v1/kv/';
    const response = await api.get(url, { params: { recurse: 'true' } });
    return response.data || [];
  } catch (error) {
    console.error('Error listing KV with values:', error);
    return [];
  }
};

// Health API
export const getHealth = async (): Promise<HealthResponse> => {
  const response = await api.get('/v1/health');
  return response.data;
};

export default api;
