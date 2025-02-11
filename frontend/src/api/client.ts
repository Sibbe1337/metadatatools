import axios, { AxiosError, AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import { ApiError, HttpMethodType, IApiResponse } from './types';

/**
 * Configuration interface for ApiClient
 */
interface IApiClientConfig {
  baseURL: string;
  timeout?: number;
  headers?: Record<string, string>;
}

/**
 * Base API client with error handling and interceptors
 */
export class ApiClient {
  private readonly client: AxiosInstance;

  constructor(config: IApiClientConfig) {
    this.client = axios.create({
      baseURL: config.baseURL,
      timeout: config.timeout || 10000,
      headers: {
        'Content-Type': 'application/json',
        ...config.headers,
      },
    });

    this.setupInterceptors();
  }

  /**
   * Sets up request and response interceptors
   */
  private setupInterceptors(): void {
    // Request interceptor
    this.client.interceptors.request.use(
      (config) => {
        // Add auth token if available
        const token = localStorage.getItem('authToken');
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor
    this.client.interceptors.response.use(
      (response) => response,
      (error: AxiosError) => {
        if (error.response) {
          const { data, status } = error.response;
          throw new ApiError(
            data.message || 'An unexpected error occurred',
            status,
            data.code,
            data.details
          );
        }
        throw new ApiError(
          error.message || 'Network error',
          0,
          'NETWORK_ERROR'
        );
      }
    );
  }

  /**
   * Makes an HTTP request
   */
  private async request<T>(
    method: HttpMethodType,
    url: string,
    config?: AxiosRequestConfig
  ): Promise<IApiResponse<T>> {
    const response: AxiosResponse<IApiResponse<T>> = await this.client.request({
      method,
      url,
      ...config,
    });
    return response.data;
  }

  /**
   * Makes a GET request
   */
  public async get<T>(url: string, config?: AxiosRequestConfig): Promise<IApiResponse<T>> {
    return this.request<T>('GET', url, config);
  }

  /**
   * Makes a POST request
   */
  public async post<T>(
    url: string,
    data?: unknown,
    config?: AxiosRequestConfig
  ): Promise<IApiResponse<T>> {
    return this.request<T>('POST', url, { ...config, data });
  }

  /**
   * Makes a PUT request
   */
  public async put<T>(
    url: string,
    data?: unknown,
    config?: AxiosRequestConfig
  ): Promise<IApiResponse<T>> {
    return this.request<T>('PUT', url, { ...config, data });
  }

  /**
   * Makes a DELETE request
   */
  public async delete<T>(url: string, config?: AxiosRequestConfig): Promise<IApiResponse<T>> {
    return this.request<T>('DELETE', url, config);
  }

  /**
   * Makes a PATCH request
   */
  public async patch<T>(
    url: string,
    data?: unknown,
    config?: AxiosRequestConfig
  ): Promise<IApiResponse<T>> {
    return this.request<T>('PATCH', url, { ...config, data });
  }

  /**
   * Uploads a file
   */
  public async uploadFile<T>(
    url: string,
    file: File,
    onProgress?: (progress: number) => void
  ): Promise<IApiResponse<T>> {
    const formData = new FormData();
    formData.append('file', file);

    return this.post<T>(url, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: (progressEvent) => {
        if (onProgress && progressEvent.total) {
          const progress = (progressEvent.loaded / progressEvent.total) * 100;
          onProgress(progress);
        }
      },
    });
  }
} 