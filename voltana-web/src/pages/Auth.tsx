import { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { register, login, resendVerification, requestOTP, verifyOTP } from '@/features/auth/api';
import { ApiError } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { InputOTP, InputOTPGroup, InputOTPSlot } from '@/components/ui/input-otp';
import { toast } from 'sonner';
import { useTranslation } from 'react-i18next';
import { Zap, MailCheck, MessageCircle } from 'lucide-react';
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

const OTP_COOLDOWN_SECS = 60;

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
  const [pendingEmail, setPendingEmail] = useState<string | null>(null);

  // OTP tab state
  const [otpPhone, setOtpPhone] = useState('');
  const [otpCode, setOtpCode] = useState('');
  const [otpSent, setOtpSent] = useState(false);
  const [otpCooldown, setOtpCooldown] = useState(0);
  const cooldownRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      if (cooldownRef.current) clearInterval(cooldownRef.current);
    };
  }, []);

  const startCooldown = () => {
    setOtpCooldown(OTP_COOLDOWN_SECS);
    cooldownRef.current = setInterval(() => {
      setOtpCooldown((s) => {
        if (s <= 1) {
          clearInterval(cooldownRef.current!);
          return 0;
        }
        return s - 1;
      });
    }, 1000);
  };

  const handleSignUp = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const validated = signUpSchema.parse(signUpData);
      await register(validated.email, validated.password);
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

  const handleOTPRequest = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!otpPhone.trim()) {
      toast.error('شماره تلفن را وارد کنید');
      return;
    }
    if (otpCooldown > 0) return;
    setLoading(true);
    try {
      await requestOTP(otpPhone.trim());
      setOtpSent(true);
      startCooldown();
      toast.success('اگر حساب متصل باشد، کد ارسال شد');
    } catch (error) {
      if (error instanceof ApiError && error.status === 429) {
        toast.error('درخواست‌های زیاد. لطفاً کمی صبر کنید');
      } else {
        toast.error('خطایی رخ داده است');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleOTPVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    if (otpCode.length !== 6) {
      toast.error('کد ۶ رقمی را وارد کنید');
      return;
    }
    setLoading(true);
    try {
      await verifyOTP(otpPhone.trim(), otpCode);
      toast.success('ورود با موفقیت انجام شد!');
      navigate('/');
    } catch (error) {
      if (error instanceof ApiError && error.code === 'INVALID_OTP') {
        toast.error('کد نامعتبر یا منقضی شده است');
      } else if (error instanceof ApiError && error.status === 429) {
        toast.error('درخواست‌های زیاد. لطفاً کمی صبر کنید');
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
              <TabsList className="grid w-full grid-cols-3">
                <TabsTrigger value="login">{t('auth.login')}</TabsTrigger>
                <TabsTrigger value="signup">{t('auth.signup')}</TabsTrigger>
                <TabsTrigger value="otp" className="flex items-center gap-1">
                  <MessageCircle className="w-3.5 h-3.5" />
                  بله/تلگرام
                </TabsTrigger>
              </TabsList>

              {/* ── Email/password login ── */}
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

              {/* ── Sign-up ── */}
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

              {/* ── OTP login via Bale/Telegram ── */}
              <TabsContent value="otp">
                <div className="space-y-4 pt-2">
                  <p className="text-sm text-muted-foreground text-center">
                    ابتدا در تنظیمات حساب خود را به بله یا تلگرام متصل کنید، سپس اینجا وارد شوید.
                  </p>

                  {!otpSent ? (
                    <form onSubmit={handleOTPRequest} className="space-y-4">
                      <div className="space-y-2">
                        <Label htmlFor="otp-phone">شماره تلفن (E.164 یا ایرانی)</Label>
                        <Input
                          id="otp-phone"
                          type="tel"
                          placeholder="+98912..."
                          value={otpPhone}
                          onChange={(e) => setOtpPhone(e.target.value)}
                          required
                          dir="ltr"
                        />
                      </div>
                      <Button
                        type="submit"
                        className="w-full"
                        disabled={loading || otpCooldown > 0}
                      >
                        {loading
                          ? 'در حال ارسال...'
                          : otpCooldown > 0
                          ? `ارسال مجدد (${otpCooldown}s)`
                          : 'ارسال کد'}
                      </Button>
                    </form>
                  ) : (
                    <form onSubmit={handleOTPVerify} className="space-y-4">
                      <p className="text-sm text-center">
                        کد ۶ رقمی ارسال‌شده به بله/تلگرام را وارد کنید:
                      </p>
                      <div className="flex justify-center">
                        <InputOTP
                          maxLength={6}
                          value={otpCode}
                          onChange={setOtpCode}
                          dir="ltr"
                        >
                          <InputOTPGroup>
                            {[0,1,2,3,4,5].map((i) => (
                              <InputOTPSlot key={i} index={i} />
                            ))}
                          </InputOTPGroup>
                        </InputOTP>
                      </div>
                      <Button
                        type="submit"
                        className="w-full"
                        disabled={loading || otpCode.length !== 6}
                      >
                        {loading ? 'در حال تأیید...' : 'تأیید و ورود'}
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        className="w-full"
                        disabled={otpCooldown > 0}
                        onClick={() => {
                          setOtpSent(false);
                          setOtpCode('');
                        }}
                      >
                        {otpCooldown > 0 ? `ارسال مجدد (${otpCooldown}s)` : 'ارسال مجدد'}
                      </Button>
                    </form>
                  )}
                </div>
              </TabsContent>
            </Tabs>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
