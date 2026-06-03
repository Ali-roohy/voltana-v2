import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { register, login, resendVerification } from '@/features/auth/api';
import { ApiError } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { toast } from 'sonner';
import { useTranslation } from 'react-i18next';
import { Zap, MailCheck } from 'lucide-react';
import { z } from 'zod';

const signUpSchema = z.object({
  email: z.string().trim().email({ message: "ایمیل نامعتبر است" }).max(255),
  password: z.string().min(8, { message: "رمز عبور باید حداقل ۸ کاراکتر باشد" }).max(100),
  full_name: z.string().trim().min(2, { message: "نام باید حداقل ۲ کاراکتر باشد" }).max(100),
  phone: z.string().trim().optional(),
});

const loginSchema = z.object({
  email: z.string().trim().email({ message: "ایمیل نامعتبر است" }).max(255),
  password: z.string().min(1, { message: "رمز عبور الزامی است" }),
});

export default function Auth() {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [signUpData, setSignUpData] = useState({
    email: '',
    password: '',
    full_name: '',
    phone: '',
  });
  const [loginData, setLoginData] = useState({
    email: '',
    password: '',
  });
  // When set, the account exists but its email is unverified — show the
  // "check your email" screen with a resend option instead of the tabs.
  const [pendingEmail, setPendingEmail] = useState<string | null>(null);

  const handleSignUp = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      const validated = signUpSchema.parse(signUpData);
      // Go API takes email+password only (full_name/phone are not persisted yet).
      await register(validated.email, validated.password);
      // Login is now gated on email verification (TASK-0009) — do NOT auto-login.
      // Show the "check your email" screen so the user can verify (or resend).
      setPendingEmail(validated.email);
      toast.success(t('auth.signupCheckEmail'));
    } catch (error) {
      if (error instanceof z.ZodError) {
        toast.error(error.errors[0].message);
      } else if (error instanceof ApiError && error.status === 409) {
        toast.error('این ایمیل قبلاً ثبت شده است');
      } else if (error instanceof ApiError) {
        toast.error(error.message || 'خطایی رخ داده است');
      } else {
        toast.error('خطایی رخ داده است');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      const validated = loginSchema.parse(loginData);

      await login(validated.email, validated.password);
      toast.success('ورود با موفقیت انجام شد!');
      navigate('/');
    } catch (error) {
      if (error instanceof z.ZodError) {
        toast.error(error.errors[0].message);
      } else if (error instanceof ApiError && error.code === 'EMAIL_NOT_VERIFIED') {
        // Account exists but email isn't verified — surface the check-email screen.
        setPendingEmail(loginData.email);
        toast.error(t('auth.loginNotVerified'));
      } else if (error instanceof ApiError && error.status === 401) {
        toast.error('ایمیل یا رمز عبور اشتباه است');
      } else if (error instanceof ApiError) {
        toast.error(error.message || 'خطایی رخ داده است');
      } else {
        toast.error('خطایی رخ داده است');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleResend = async () => {
    if (!pendingEmail) return;
    setLoading(true);
    try {
      await resendVerification(pendingEmail);
      toast.success(t('auth.resendSuccess'));
    } catch (error) {
      if (error instanceof ApiError && error.status === 429) {
        toast.error(t('auth.resendThrottled'));
      } else {
        toast.error('خطایی رخ داده است');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-background via-secondary to-background p-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-primary mb-4 shadow-glow">
            <Zap className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-4xl font-bold bg-gradient-primary bg-clip-text text-transparent mb-2">
            {t('app.name')}
          </h1>
          <p className="text-muted-foreground">{t('app.tagline')}</p>
        </div>

        <Card className="shadow-soft">
          <CardHeader>
            <CardTitle className="text-2xl text-center">خوش آمدید</CardTitle>
            <CardDescription className="text-center">
              برای شروع وارد شوید یا ثبت نام کنید
            </CardDescription>
          </CardHeader>
          <CardContent>
            {pendingEmail ? (
              <div className="text-center space-y-4 py-2">
                <MailCheck className="w-12 h-12 text-primary mx-auto" />
                <div className="space-y-1">
                  <p>{t('auth.checkEmail')}</p>
                  <p className="font-medium" dir="ltr">{pendingEmail}</p>
                </div>
                <p className="text-sm text-muted-foreground">{t('auth.checkEmailHint')}</p>
                <Button className="w-full" onClick={handleResend} disabled={loading}>
                  {loading ? '...' : t('auth.resend')}
                </Button>
                <Button variant="ghost" className="w-full" onClick={() => setPendingEmail(null)}>
                  {t('auth.backToLogin')}
                </Button>
              </div>
            ) : (
            <Tabs defaultValue="login" dir="rtl">
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="login">{t('auth.login')}</TabsTrigger>
                <TabsTrigger value="signup">{t('auth.signup')}</TabsTrigger>
              </TabsList>

              <TabsContent value="login">
                <form onSubmit={handleLogin} className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="login-email">{t('auth.email')}</Label>
                    <Input
                      id="login-email"
                      type="email"
                      value={loginData.email}
                      onChange={(e) => setLoginData({ ...loginData, email: e.target.value })}
                      required
                      dir="ltr"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="login-password">{t('auth.password')}</Label>
                    <Input
                      id="login-password"
                      type="password"
                      value={loginData.password}
                      onChange={(e) => setLoginData({ ...loginData, password: e.target.value })}
                      required
                      dir="ltr"
                    />
                  </div>
                  <Button type="submit" className="w-full" disabled={loading}>
                    {loading ? 'در حال ورود...' : t('auth.login')}
                  </Button>
                </form>
              </TabsContent>

              <TabsContent value="signup">
                <form onSubmit={handleSignUp} className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="signup-name">{t('auth.name')}</Label>
                    <Input
                      id="signup-name"
                      type="text"
                      value={signUpData.full_name}
                      onChange={(e) => setSignUpData({ ...signUpData, full_name: e.target.value })}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="signup-email">{t('auth.email')}</Label>
                    <Input
                      id="signup-email"
                      type="email"
                      value={signUpData.email}
                      onChange={(e) => setSignUpData({ ...signUpData, email: e.target.value })}
                      required
                      dir="ltr"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="signup-phone">{t('auth.phone')} (اختیاری)</Label>
                    <Input
                      id="signup-phone"
                      type="tel"
                      value={signUpData.phone}
                      onChange={(e) => setSignUpData({ ...signUpData, phone: e.target.value })}
                      dir="ltr"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="signup-password">{t('auth.password')}</Label>
                    <Input
                      id="signup-password"
                      type="password"
                      value={signUpData.password}
                      onChange={(e) => setSignUpData({ ...signUpData, password: e.target.value })}
                      required
                      dir="ltr"
                    />
                  </div>
                  <Button type="submit" className="w-full" disabled={loading}>
                    {loading ? 'در حال ثبت نام...' : t('auth.signup')}
                  </Button>
                </form>
              </TabsContent>
            </Tabs>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
