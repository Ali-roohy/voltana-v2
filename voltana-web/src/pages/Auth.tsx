import { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { register, login, resendVerification, requestOTP, verifyOTP, registerWithOTP } from '@/features/auth/api';
import { ApiError } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { OTPInput6 } from '@/components/ui/otp-input';
import { toast } from 'sonner';
import { useTranslation } from 'react-i18next';
import { Zap, MailCheck, Mail, MessageCircle, Send } from 'lucide-react';
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

const OTP_TTL_SECS = 60;
const OTP_COOLDOWN_SECS = 60;

type OTPError =
  | { type: 'invalid'; remaining: number }
  | { type: 'locked' }
  | { type: 'phone_taken' }
  | null;

// ── BotOTPTab ──────────────────────────────────────────────────────────────────
// Shared component for both OTP login (mode="login") and OTP registration (mode="register").

interface BotOTPTabProps {
  platform: 'bale' | 'telegram';
  mode: 'login' | 'register';
  onBack?: () => void; // register mode: returns to method picker
}

function BotOTPTab({ platform, mode, onBack }: BotOTPTabProps) {
  const navigate = useNavigate();
  const platformLabel = platform === 'bale' ? 'بله' : 'تلگرام';

  const [phone, setPhone] = useState('');
  const [email, setEmail] = useState('');
  const [code, setCode] = useState('');
  const [step, setStep] = useState<'phone' | 'otp'>('phone');
  const [loading, setLoading] = useState(false);
  const [otpTimer, setOtpTimer] = useState(0);
  const [cooldown, setCooldown] = useState(0);
  const [otpError, setOtpError] = useState<OTPError>(null);

  const otpTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const cooldownRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      if (otpTimerRef.current) clearInterval(otpTimerRef.current);
      if (cooldownRef.current) clearInterval(cooldownRef.current);
    };
  }, []);

  const startTimers = () => {
    setOtpTimer(OTP_TTL_SECS);
    if (otpTimerRef.current) clearInterval(otpTimerRef.current);
    otpTimerRef.current = setInterval(() => {
      setOtpTimer((s) => {
        if (s <= 1) { clearInterval(otpTimerRef.current!); return 0; }
        return s - 1;
      });
    }, 1000);

    setCooldown(OTP_COOLDOWN_SECS);
    if (cooldownRef.current) clearInterval(cooldownRef.current);
    cooldownRef.current = setInterval(() => {
      setCooldown((s) => {
        if (s <= 1) { clearInterval(cooldownRef.current!); return 0; }
        return s - 1;
      });
    }, 1000);
  };

  const sendOTP = async (phoneVal: string) => {
    // Connectivity reminder — fires before every OTP API call.
    toast.warning("فیلترشکن رو روشن کن برای تلگرام، یا بله رو چک کن");
    setLoading(true);
    try {
      await requestOTP(phoneVal, platform);
      setStep('otp');
      setCode('');
      setOtpError(null);
      startTimers();
    } catch (err) {
      if (err instanceof ApiError && err.status === 429) {
        toast.error('درخواست زیاد — لطفاً ۱۵ دقیقه دیگر امتحان کنید.');
      } else {
        toast.error('خطایی رخ داده است');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleRequest = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!phone.trim() || loading || cooldown > 0) return;
    await sendOTP(phone.trim());
  };

  const handleResend = async () => {
    if (!phone.trim() || loading || cooldown > 0) return;
    await sendOTP(phone.trim());
  };

  const doVerify = useCallback(async (codeVal: string) => {
    if (codeVal.length !== 6 || loading || otpTimer === 0) return;
    setLoading(true);
    try {
      if (mode === 'register') {
        await registerWithOTP(phone.trim(), codeVal, platform, email.trim() || undefined);
        toast.success('ثبت نام با موفقیت انجام شد!');
      } else {
        await verifyOTP(phone.trim(), codeVal, platform);
        toast.success('ورود با موفقیت انجام شد!');
      }
      navigate('/');
    } catch (err) {
      setCode('');
      if (err instanceof ApiError && err.status === 409 && err.code === 'PHONE_TAKEN') {
        setOtpError({ type: 'phone_taken' });
      } else if (err instanceof ApiError && err.status === 401) {
        if (err.code === 'OTP_LOCKED') {
          setOtpError({ type: 'locked' });
        } else {
          const remaining = (err.data?.remaining_attempts as number) ?? 0;
          setOtpError({ type: 'invalid', remaining });
        }
      } else if (err instanceof ApiError && err.status === 429) {
        toast.error('درخواست‌های زیاد. لطفاً کمی صبر کنید');
      } else {
        toast.error('خطایی رخ داده است');
      }
    } finally {
      setLoading(false);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loading, otpTimer, phone, email, mode, platform]);

  const isExpired = step === 'otp' && otpTimer === 0;
  const isLocked = otpError?.type === 'locked';
  const isPhoneTaken = otpError?.type === 'phone_taken';

  const helperText = mode === 'register'
    ? `ابتدا ربات ولتانا را در ${platformLabel} استارت کنید و شماره خود را با ربات به اشتراک بگذارید.`
    : `ابتدا باید در تنظیمات حساب کاربری، شماره تلفن خود را به ${platformLabel} متصل کنید.`;

  return (
    <div className="space-y-4 pt-2">
      {step === 'phone' ? (
        <form onSubmit={handleRequest} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor={`${platform}-${mode}-phone`}>شماره تلفن</Label>
            <Input
              id={`${platform}-${mode}-phone`}
              type="tel"
              placeholder="۰۹۱۲ ۳۴۵ ۶۷۸۹"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              required
              dir="ltr"
            />
            <p className="text-xs text-muted-foreground">{helperText}</p>
          </div>
          {mode === 'register' && (
            <div className="space-y-2">
              <Label htmlFor={`${platform}-reg-email`}>ایمیل (اختیاری)</Label>
              <Input
                id={`${platform}-reg-email`}
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                dir="ltr"
              />
            </div>
          )}
          <Button type="submit" className="w-full" disabled={loading || cooldown > 0}>
            {loading ? 'در حال ارسال...' : cooldown > 0 ? `ارسال مجدد (${cooldown}ث)` : 'ارسال کد'}
          </Button>
          {mode === 'register' && onBack && (
            <Button type="button" variant="ghost" className="w-full" onClick={onBack}>
              بازگشت
            </Button>
          )}
        </form>
      ) : (
        <div className="space-y-4">
          <p className="text-sm text-center text-muted-foreground">
            کد ۶ رقمی ارسال‌شده به {platformLabel} را وارد کنید:
          </p>

          <OTPInput6
            value={code}
            onChange={(val) => { setCode(val); setOtpError(null); }}
            onComplete={doVerify}
            loading={loading}
            disabled={isExpired || isLocked || isPhoneTaken}
          />

          {!isExpired && !isLocked && !isPhoneTaken && !loading && (
            <p className="text-xs text-center text-muted-foreground">
              کد تا {otpTimer} ثانیه دیگر معتبر است
            </p>
          )}

          {isLocked && (
            <p className="text-sm text-center text-destructive font-medium">
              حساب قفل شده — ۱۵ دقیقه دیگر تلاش کنید.
            </p>
          )}
          {isExpired && !isLocked && (
            <p className="text-sm text-center text-destructive font-medium">
              کد منقضی شد — دوباره ارسال کنید
            </p>
          )}
          {otpError?.type === 'invalid' && !isExpired && (
            <p className="text-sm text-center text-destructive">
              کد اشتباه است — {otpError.remaining} تلاش باقی مانده
            </p>
          )}
          {isPhoneTaken && (
            <p className="text-sm text-center text-destructive">
              این شماره قبلاً ثبت شده است — وارد شوید
            </p>
          )}

          <Button
            type="button"
            variant="ghost"
            className="w-full"
            disabled={loading || (cooldown > 0 && !isExpired) || isLocked}
            onClick={handleResend}
          >
            {cooldown > 0 && !isExpired ? `ارسال مجدد (${cooldown}ث)` : 'ارسال مجدد'}
          </Button>
        </div>
      )}
    </div>
  );
}

// ── EmailRegisterStep ──────────────────────────────────────────────────────────

interface EmailRegisterStepProps {
  onBack: () => void;
  onPendingEmail: (email: string) => void;
}

function EmailRegisterStep({ onBack, onPendingEmail }: EmailRegisterStepProps) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [signUpData, setSignUpData] = useState({ email: '', password: '', full_name: '', phone: '' });

  const handleSignUp = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const validated = signUpSchema.parse(signUpData);
      await register(validated.email, validated.password);
      onPendingEmail(validated.email);
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

  return (
    <div className="space-y-4 pt-2">
      <Button type="button" variant="ghost" size="sm" className="mb-1 px-0" onClick={onBack}>
        ← بازگشت
      </Button>
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
    </div>
  );
}

// ── RegisterFlow ───────────────────────────────────────────────────────────────

type RegisterStep = 'picker' | 'email' | 'bale' | 'telegram';

interface RegisterFlowProps {
  onPendingEmail: (email: string) => void;
}

function RegisterFlow({ onPendingEmail }: RegisterFlowProps) {
  const [step, setStep] = useState<RegisterStep>('picker');

  if (step === 'email') {
    return <EmailRegisterStep onBack={() => setStep('picker')} onPendingEmail={onPendingEmail} />;
  }
  if (step === 'bale') {
    return <BotOTPTab platform="bale" mode="register" onBack={() => setStep('picker')} />;
  }
  if (step === 'telegram') {
    return <BotOTPTab platform="telegram" mode="register" onBack={() => setStep('picker')} />;
  }

  return (
    <div className="space-y-4 pt-2">
      <p className="text-sm text-center text-muted-foreground">روش ثبت نام را انتخاب کنید</p>
      <div className="grid grid-cols-3 gap-3">
        {([
          { id: 'email' as RegisterStep, icon: Mail, label: 'ایمیل' },
          { id: 'bale' as RegisterStep, icon: MessageCircle, label: 'بله' },
          { id: 'telegram' as RegisterStep, icon: Send, label: 'تلگرام' },
        ] as const).map(({ id, icon: Icon, label }) => (
          <Card
            key={id}
            className="cursor-pointer hover:border-primary transition-colors"
            onClick={() => setStep(id)}
          >
            <CardContent className="py-4 text-center">
              <Icon className="w-6 h-6 mx-auto mb-2 text-primary" />
              <p className="text-sm font-medium">{label}</p>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}

// ── Auth page ──────────────────────────────────────────────────────────────────

export default function Auth() {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [loginData, setLoginData] = useState({ email: '', password: '' });
  const [pendingEmail, setPendingEmail] = useState<string | null>(null);
  const [mode, setMode] = useState<'login' | 'register'>('login');

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
              <>
                {/* Mode toggle */}
                <div className="flex gap-2 mb-4">
                  <Button
                    variant={mode === 'login' ? 'default' : 'ghost'}
                    className="flex-1"
                    onClick={() => setMode('login')}
                  >
                    ورود
                  </Button>
                  <Button
                    variant={mode === 'register' ? 'default' : 'ghost'}
                    className="flex-1"
                    onClick={() => setMode('register')}
                  >
                    ثبت نام
                  </Button>
                </div>

                {mode === 'login' ? (
                  // ── Login: three-tab (ایمیل | بله | تلگرام) ──────────────────
                  <Tabs key="login" defaultValue="email" dir="rtl">
                    <TabsList className="grid w-full grid-cols-3">
                      <TabsTrigger value="email">ایمیل</TabsTrigger>
                      <TabsTrigger value="bale">بله</TabsTrigger>
                      <TabsTrigger value="telegram">تلگرام</TabsTrigger>
                    </TabsList>

                    <TabsContent value="email">
                      <form onSubmit={handleLogin} className="space-y-4 pt-2">
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

                    <TabsContent value="bale">
                      <BotOTPTab platform="bale" mode="login" />
                    </TabsContent>

                    <TabsContent value="telegram">
                      <BotOTPTab platform="telegram" mode="login" />
                    </TabsContent>
                  </Tabs>
                ) : (
                  // ── Register: method picker → step 2 ─────────────────────────
                  <RegisterFlow key="register" onPendingEmail={setPendingEmail} />
                )}
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
