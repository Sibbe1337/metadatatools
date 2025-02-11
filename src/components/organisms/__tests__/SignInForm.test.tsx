import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../../../contexts/AuthContext';
import { SignInForm } from '../SignInForm';
import { authService } from '../../../services/authService';
import { useNotifications } from '../../../contexts/UIContext';

jest.mock('../../../services/authService');
jest.mock('../../../contexts/UIContext');

const mockLogin = authService.login as jest.Mock;
const mockGetCurrentUser = authService.getCurrentUser as jest.Mock;
const mockShowError = jest.fn();

beforeEach(() => {
  mockLogin.mockReset();
  mockGetCurrentUser.mockReset();
  (useNotifications as jest.Mock).mockReturnValue({ showError: mockShowError });
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
  it('shows loading state when submitting', () => {
    // Mock useAuth with loading state
    jest.spyOn(require('../../../contexts/AuthContext'), 'useAuth').mockImplementation(() => ({
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
}); 