import { chromium } from "playwright";

const BASE = "http://127.0.0.1:4173";
const ts = Date.now();
const seeded = { email: `qa_seed_${ts}@v.test`, pw: "abcd1234" };
const fresh = { email: `qa_fresh_${ts}@v.test`, pw: "abcd1234" };

const results = [];
const ok = (s) => { results.push(["PASS", s]); console.log("  ✓", s); };
const bad = (s, e) => { results.push(["FAIL", s + (e ? ` — ${e}` : "")]); console.log("  ✗", s, e ?? ""); };
async function step(name, fn) { try { await fn(); ok(name); } catch (e) { bad(name, e.message?.split("\n")[0]); } }

async function api(path, method = "GET", body, token) {
  const r = await fetch(BASE + path, {
    method,
    headers: { "content-type": "application/json", ...(token ? { authorization: `Bearer ${token}` } : {}) },
    body: body ? JSON.stringify(body) : undefined,
  });
  return { status: r.status, body: await r.json().catch(() => null) };
}

// ── seed a user with data via the API (so authenticated pages have something to render)
async function seed() {
  await api("/auth/register", "POST", seeded);
  const { body } = await api("/auth/login", "POST", seeded);
  const t = body.access_token;
  await api("/v1/settings", "PUT", { default_car_id: null, peak_rate: 12, mid_rate: 6, offpeak_rate: 3 }, t);
  const car = await api("/v1/cars", "POST", { name: "Seeded EV" }, t);
  const carId = car.body.id;
  await api("/v1/settings", "PUT", { default_car_id: carId, peak_rate: 12, mid_rate: 6, offpeak_rate: 3 }, t);
  await api("/v1/charging-sessions", "POST", {
    car_id: carId, started_at: "2026-05-31T08:00:00.000Z", ended_at: "2026-05-31T09:30:00.000Z",
    energy_peak_kwh: 2, energy_mid_kwh: 3, energy_offpeak_kwh: 4, start_soc: 30, end_soc: 80,
  }, t);
  console.log("seeded user + car + session + rates");
}

const consoleErrors = [];
const netErrors = [];

(async () => {
  await seed();
  const browser = await chromium.launch();
  const page = await browser.newPage();
  page.on("console", (m) => { if (m.type() === "error") consoleErrors.push(m.text()); });
  page.on("pageerror", (e) => consoleErrors.push("PAGEERROR: " + e.message));
  page.on("response", (r) => {
    const u = r.url();
    if (/\/(auth|v1)\//.test(u) && r.status() >= 400) netErrors.push(`${r.status()} ${r.request().method()} ${new URL(u).pathname}`);
  });

  // ── B) login as seeded user → render authenticated pages
  await step("app loads at /auth", async () => {
    await page.goto(`${BASE}/auth`, { waitUntil: "networkidle" });
    await page.waitForSelector("#login-email", { timeout: 8000 });
  });
  await step("login (seeded user) → dashboard", async () => {
    await page.fill("#login-email", seeded.email);
    await page.fill("#login-password", seeded.pw);
    await page.click('form:has(#login-password) button[type="submit"]');
    await page.waitForFunction(() => location.pathname === "/", { timeout: 8000 });
  });
  await step("cars page renders seeded car", async () => {
    await page.goto(`${BASE}/cars`, { waitUntil: "networkidle" });
    await page.getByText("Seeded EV").first().waitFor({ timeout: 8000 });
  });
  await step("charging page renders seeded session (car name)", async () => {
    await page.goto(`${BASE}/charging`, { waitUntil: "networkidle" });
    await page.getByText("Seeded EV").first().waitFor({ timeout: 8000 });
  });
  await step("settings page shows persisted peak rate = 12", async () => {
    await page.goto(`${BASE}/settings`, { waitUntil: "networkidle" });
    const v = await page.inputValue("#ratePeak");
    if (v !== "12") throw new Error(`#ratePeak = ${v}, expected 12`);
  });
  await step("logout → returns to /auth", async () => {
    await page.getByRole("button", { name: /خروج|logout/i }).click();
    await page.waitForFunction(() => location.pathname === "/auth", { timeout: 8000 });
  });

  // ── A) signup a fresh user → auto-login → dashboard
  await step("signup (fresh user, 8-char pw) → dashboard", async () => {
    await page.goto(`${BASE}/auth`, { waitUntil: "networkidle" });
    await page.getByRole("tab").nth(1).click(); // signup tab
    await page.fill("#signup-name", "QA Tester");
    await page.fill("#signup-email", fresh.email);
    await page.fill("#signup-password", fresh.pw);
    await page.click('form:has(#signup-password) button[type="submit"]');
    await page.waitForFunction(() => location.pathname === "/", { timeout: 8000 });
  });
  await step("UI add car → appears in list", async () => {
    await page.goto(`${BASE}/cars`, { waitUntil: "networkidle" });
    await page.getByRole("button", { name: /افزودن|add car/i }).first().click();
    await page.fill("#name", "Browser Car");
    await page.getByRole("button", { name: /ذخیره|save/i }).first().click();
    await page.getByText("Browser Car").first().waitFor({ timeout: 8000 });
  });

  await browser.close();

  console.log("\n=== CONSOLE ERRORS ===");
  console.log(consoleErrors.length ? consoleErrors.join("\n") : "  none");
  console.log("=== NETWORK 4xx/5xx on /auth,/v1 ===");
  console.log(netErrors.length ? netErrors.join("\n") : "  none");
  const fails = results.filter((r) => r[0] === "FAIL").length;
  console.log(`\n=== SUMMARY: ${results.length - fails}/${results.length} steps passed ===`);
  process.exit(fails ? 1 : 0);
})().catch((e) => { console.error("FATAL", e); process.exit(2); });
