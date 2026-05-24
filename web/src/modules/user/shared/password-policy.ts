const MINIMUM_PASSWORD_LENGTH = 12;

export type PasswordPolicyEvaluation = {
  meetsMinimum: boolean;
  checks: {
    minLength: boolean;
    hasLetter: boolean;
    hasDigit: boolean;
  };
};

export function evaluateUserPasswordPolicy(password: string): PasswordPolicyEvaluation {
  const hasLetter = /[A-Za-z]/.test(password);
  const hasDigit = /\d/.test(password);
  const minLength = password.length >= MINIMUM_PASSWORD_LENGTH;
  const meetsMinimum = minLength && hasLetter && hasDigit;

  return {
    meetsMinimum,
    checks: {
      minLength,
      hasLetter,
      hasDigit,
    },
  };
}
