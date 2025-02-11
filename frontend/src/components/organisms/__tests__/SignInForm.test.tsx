import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../../../context/AuthContext';
import { SignInForm } from '../SignInForm';
import { authService } from '../../../services/authService';
import * as UIContext from '../../../context/UIContext';
import { AuthErrorType } from '../../../types/auth';

// Mock environment variables
jest.mock('../../../services/authService', () => ({
  authService: {
    login: jest.fn(),
    getCurrentUser: jest.fn(),
  },
}));

jest.mock('../../../context/UIContext');

const mockLogin = authService.login as jest.Mock;
const mockGetCurrentUser = authService.getCurrentUser as jest.Mock;
const mockShowError = jest.fn();

beforeEach(() => {
  mockLogin.mockReset();
  mockGetCurrentUser.mockReset();
  jest.spyOn(UIContext, 'useNotifications').mockReturnValue({
    notifications: [],
    modalState: { isOpen: false },
    showNotification: jest.fn(),
    hideNotification: jest.fn(),
    removeNotification: jest.fn(),
    showError: mockShowError,
    showSuccess: jest.fn(),
    showInfo: jest.fn(),
    showWarning: jest.fn(),
    showModal: jest.fn(),
    hideModal: jest.fn(),
  });
});

const renderSignInForm = () => {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <SignInForm />
      </AuthProvider>
    </MemoryRouter>
  );
};

describe('SignInForm', () => {
  it('renders all form elements correctly', () => {
    renderSignInForm();

    expect(screen.getByText('Sign in to your account')).toBeInTheDocument();
    expect(screen.getByTestId('email-input')).toBeInTheDocument();
    expect(screen.getByTestId('password-input')).toBeInTheDocument();
    expect(screen.getByTestId('remember-me-checkbox')).toBeInTheDocument();
    expect(screen.getByTestId('submit-button')).toBeInTheDocument();
  });

  it('shows loading state when submitting', () => {
    // Mock useAuth with loading state
    jest.spyOn(require('../../../context/AuthContext'), 'useAuth').mockImplementation(() => ({
      login: jest.fn(),
      error: null,
      isLoading: true,
    }));

    renderSignInForm();

    const submitButton = screen.getByTestId('submit-button');
    expect(submitButton).toBeDisabled();
    expect(submitButton).toHaveClass('disabled:opacity-50');
    
    const spinner = screen.getByTestId('loading-spinner');
    expect(spinner).toBeInTheDocument();
  });

  it('displays validation errors for invalid input', async () => {
    // Mock useAuth with non-loading state
    jest.spyOn(require('../../../context/AuthContext'), 'useAuth').mockImplementation(() => ({
      login: jest.fn(),
      error: null,
      isLoading: false,
    }));

    renderSignInForm();

    const emailInput = screen.getByTestId('email-input');
    const passwordInput = screen.getByTestId('password-input');
    const form = screen.getByRole('form');

    // Submit form with invalid fields
    fireEvent.change(emailInput, { target: { value: 'invalid-email' } });
    fireEvent.change(passwordInput, { target: { value: 'short' } });
    fireEvent.submit(form);

    // Wait for validation errors
    await waitFor(() => {
      const alerts = screen.getAllByRole('alert');
      expect(alerts).toHaveLength(2);
      expect(alerts[0]).toHaveTextContent('Please enter a valid email address');
      expect(alerts[1]).toHaveTextContent('Password must be at least 8 characters');
    });
  });

  it('displays error message when authentication fails', () => {
    const error = {
      type: AuthErrorType.INVALID_CREDENTIALS,
      message: 'Invalid email or password',
    };

    jest.spyOn(require('../../../context/AuthContext'), 'useAuth').mockImplementation(() => ({
      login: jest.fn(),
      error,
      isLoading: false,
    }));

    renderSignInForm();
    expect(screen.getByText('Invalid email or password')).toBeInTheDocument();
  });

  it('calls login function with correct data on valid submission', async () => {
    const mockLogin = jest.fn();
    
    jest.spyOn(require('../../../context/AuthContext'), 'useAuth').mockImplementation(() => ({
      login: mockLogin,
      error: null,
      isLoading: false,
    }));

    renderSignInForm();

    const emailInput = screen.getByTestId('email-input');
    const passwordInput = screen.getByTestId('password-input');
    const form = screen.getByRole('form');

    fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
    fireEvent.change(passwordInput, { target: { value: 'password123' } });
    fireEvent.submit(form);

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123',
      });
    });
  });
}); 