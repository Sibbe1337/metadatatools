import { render, screen } from '@testing-library/react';
import { FormField } from '../FormField';
import { AuthInput } from '../../atoms/AuthInput';

describe('FormField', () => {
  it('renders children correctly', () => {
    render(
      <FormField>
        <AuthInput name="test" placeholder="Test input" />
      </FormField>
    );
    expect(screen.getByPlaceholderText('Test input')).toBeInTheDocument();
  });

  it('renders label when provided', () => {
    render(
      <FormField label="Test Label">
        <AuthInput name="test" />
      </FormField>
    );
    expect(screen.getByText('Test Label')).toBeInTheDocument();
  });

  it('shows required indicator when required prop is true', () => {
    render(
      <FormField label="Test Label" required>
        <AuthInput name="test" />
      </FormField>
    );
    expect(screen.getByText('*')).toBeInTheDocument();
  });

  it('displays error message when error prop is provided', () => {
    const errorMessage = 'This field is required';
    render(
      <FormField error={errorMessage}>
        <AuthInput name="test" />
      </FormField>
    );
    expect(screen.getByRole('alert')).toHaveTextContent(errorMessage);
  });

  it('displays helper text when provided and no error', () => {
    const helperText = 'Helper text';
    render(
      <FormField helperText={helperText}>
        <AuthInput name="test" />
      </FormField>
    );
    expect(screen.getByText(helperText)).toBeInTheDocument();
  });

  it('prioritizes error message over helper text', () => {
    const helperText = 'Helper text';
    const errorMessage = 'Error message';
    render(
      <FormField helperText={helperText} error={errorMessage}>
        <AuthInput name="test" />
      </FormField>
    );
    expect(screen.getByText(errorMessage)).toBeInTheDocument();
    expect(screen.queryByText(helperText)).not.toBeInTheDocument();
  });

  it('applies custom className to wrapper', () => {
    const customClass = 'custom-wrapper';
    render(
      <FormField className={customClass}>
        <AuthInput name="test" />
      </FormField>
    );
    expect(screen.getByTestId('form-field')).toHaveClass(customClass);
  });

  it('applies custom labelClassName to label', () => {
    const customLabelClass = 'custom-label';
    render(
      <FormField label="Test Label" labelClassName={customLabelClass}>
        <AuthInput name="test" />
      </FormField>
    );
    expect(screen.getByText('Test Label')).toHaveClass(customLabelClass);
  });
}); 