import React, { createContext, useContext, useState, useCallback } from 'react';
import { v4 as uuidv4 } from 'uuid';

type NotificationType = 'success' | 'error' | 'warning' | 'info';

interface Notification {
  id: string;
  type: NotificationType;
  message: string;
}

interface ModalState {
  isOpen: boolean;
  title?: string;
  message?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  onConfirm?: () => void;
}

interface UIContextType {
  notifications: Notification[];
  modalState: ModalState;
  showNotification: (type: NotificationType, message: string, duration?: number) => void;
  hideNotification: (id: string) => void;
  showModal: (options: Omit<ModalState, 'isOpen'>) => void;
  hideModal: () => void;
  showSuccess: (message: string) => void;
  showError: (message: string) => void;
  showInfo: (message: string) => void;
  showWarning: (message: string) => void;
  removeNotification: (id: string) => void;
}

const UIContext = createContext<UIContextType | null>(null);

export const UIProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [modalState, setModalState] = useState<ModalState>({
    isOpen: false,
  });

  const showNotification = useCallback(
    (type: NotificationType, message: string, duration = 5000) => {
      const id = uuidv4();
      const notification = { id, type, message };

      setNotifications((prev) => [...prev, notification]);

      if (duration > 0) {
        setTimeout(() => {
          hideNotification(id);
        }, duration);
      }
    },
    []
  );

  const hideNotification = useCallback((id: string) => {
    setNotifications((prev) => prev.filter((notification) => notification.id !== id));
  }, []);

  const showModal = useCallback((options: Omit<ModalState, 'isOpen'>) => {
    setModalState({ isOpen: true, ...options });
  }, []);

  const hideModal = useCallback(() => {
    setModalState({ isOpen: false });
  }, []);

  const addNotification = useCallback((message: string, type: Notification['type']) => {
    const id = Math.random().toString(36).substring(7);
    setNotifications(prev => [...prev, { id, message, type }]);

    // Auto-remove notification after 5 seconds
    setTimeout(() => {
      removeNotification(id);
    }, 5000);
  }, []);

  const removeNotification = useCallback((id: string) => {
    setNotifications(prev => prev.filter(notification => notification.id !== id));
  }, []);

  const showSuccess = useCallback((message: string) => {
    addNotification(message, 'success');
  }, [addNotification]);

  const showError = useCallback((message: string) => {
    addNotification(message, 'error');
  }, [addNotification]);

  const showInfo = useCallback((message: string) => {
    addNotification(message, 'info');
  }, [addNotification]);

  const showWarning = useCallback((message: string) => {
    addNotification(message, 'warning');
  }, [addNotification]);

  const value = {
    notifications,
    modalState,
    showNotification,
    hideNotification,
    showModal,
    hideModal,
    showSuccess,
    showError,
    showInfo,
    showWarning,
    removeNotification,
  };

  return <UIContext.Provider value={value}>{children}</UIContext.Provider>;
};

export const useUI = () => {
  const context = useContext(UIContext);
  if (context === undefined) {
    throw new Error('useUI must be used within a UIProvider');
  }
  return context;
};

export const useNotifications = () => {
  const context = useContext(UIContext);
  if (!context) {
    throw new Error('useNotifications must be used within a UIProvider');
  }
  return context;
}; 