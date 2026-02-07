const { chromium } = require('playwright');

const BASE_URL = 'http://192.168.1.145:3002';
const API_URL = 'http://192.168.1.145:3001';

async function comprehensiveTest() {
  console.log('🔬 COMPREHENSIVE DOCKERVERSE TEST SUITE\n');
  console.log('=' .repeat(60));
  
  const browser = await chromium.launch({ headless: false, slowMo: 400 });
  const context = await browser.newContext({ 
    viewport: { width: 1400, height: 900 },
    ignoreHTTPSErrors: true
  });
  const page = await context.newPage();

  // Capture console logs
  page.on('console', msg => {
    if (msg.type() === 'error') {
      console.log(`  🔴 Console Error: ${msg.text()}`);
    }
  });
  
  page.on('requestfailed', req => {
    console.log(`  🔴 Request failed: ${req.url()} - ${req.failure()?.errorText}`);
  });

  const results = [];
  let authToken = null;

  try {
    // ═══════════════════════════════════════════════════════════
    // TEST 1: LOGIN
    // ═══════════════════════════════════════════════════════════
    console.log('\n' + '═'.repeat(60));
    console.log('TEST 1: LOGIN AS ADMIN');
    console.log('═'.repeat(60));
    
    await page.goto(BASE_URL, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await page.screenshot({ path: 'test-screenshots/comp-01-login-page.png', fullPage: true });
    
    // Login
    await page.fill('input[type="text"], input[placeholder*="user"], input[autocomplete="username"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(3000);
    
    // Get token from localStorage
    authToken = await page.evaluate(() => localStorage.getItem('auth_access_token'));
    console.log(`  Auth token obtained: ${authToken ? 'YES' : 'NO'}`);
    
    const onDashboard = await page.locator('h1:has-text("DockerVerse")').isVisible();
    results.push({ test: 'Login', passed: onDashboard && authToken });
    console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ FAILED');
    
    await page.screenshot({ path: 'test-screenshots/comp-01-dashboard.png', fullPage: true });

    // ═══════════════════════════════════════════════════════════
    // TEST 2: THEME TOGGLE IN HEADER
    // ═══════════════════════════════════════════════════════════
    console.log('\n' + '═'.repeat(60));
    console.log('TEST 2: THEME TOGGLE IN TOP MENU (User Request)');
    console.log('═'.repeat(60));
    
    // Check if theme button exists in header
    const themeButtonInHeader = page.locator('header button:has(svg[class*="Moon"]), header button:has(svg[class*="Sun"])').first();
    const themeInHeaderExists = await themeButtonInHeader.isVisible().catch(() => false);
    console.log(`  Theme button in header: ${themeInHeaderExists ? 'EXISTS' : 'NOT FOUND - NEEDS TO BE ADDED'}`);
    
    results.push({ test: 'Theme in Header', passed: themeInHeaderExists });
    console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ NEEDS FIX - Add theme toggle to header');

    // ═══════════════════════════════════════════════════════════
    // TEST 3: OPEN SETTINGS AND CHECK PROFILE SAVE
    // ═══════════════════════════════════════════════════════════
    console.log('\n' + '═'.repeat(60));
    console.log('TEST 3: PROFILE SAVE BUTTON');
    console.log('═'.repeat(60));
    
    // Open user menu
    await page.locator('.user-menu-container button').first().click();
    await page.waitForTimeout(500);
    
    // Click Settings
    const settingsBtn = page.locator('button:has-text("Settings"), button:has-text("Configuración")').first();
    await settingsBtn.click();
    await page.waitForTimeout(800);
    
    // Click Profile
    const profileBtn = page.locator('button:has-text("My Profile"), button:has-text("Mi Perfil")').first();
    if (await profileBtn.isVisible()) {
      await profileBtn.click();
      await page.waitForTimeout(500);
    }
    
    await page.screenshot({ path: 'test-screenshots/comp-03-profile.png', fullPage: true });
    
    // Check for Save button
    const profileSaveBtn = page.locator('.fixed button:has-text("Save"), .fixed button:has-text("Guardar")').first();
    const profileSaveExists = await profileSaveBtn.isVisible().catch(() => false);
    console.log(`  Profile Save button visible: ${profileSaveExists}`);
    
    results.push({ test: 'Profile Save Button', passed: profileSaveExists });
    console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ FAILED');

    // ═══════════════════════════════════════════════════════════
    // TEST 4: PASSWORD CHANGE WITH FEEDBACK
    // ═══════════════════════════════════════════════════════════
    console.log('\n' + '═'.repeat(60));
    console.log('TEST 4: PASSWORD CHANGE WITH SUCCESS MESSAGE');
    console.log('═'.repeat(60));
    
    // Go back to main settings
    const backBtn = page.locator('.fixed button:has-text("Back"), .fixed button:has-text("Volver")').first();
    if (await backBtn.isVisible()) {
      await backBtn.click();
      await page.waitForTimeout(500);
    }
    
    // Click Password
    const passwordBtn = page.locator('button:has-text("Change Password"), button:has-text("Cambiar Contraseña")').first();
    if (await passwordBtn.isVisible()) {
      await passwordBtn.click();
      await page.waitForTimeout(500);
    }
    
    await page.screenshot({ path: 'test-screenshots/comp-04-password.png', fullPage: true });
    
    // Fill password form (note: we won't actually change it to avoid breaking tests)
    const currentPwdInput = page.locator('input[type="password"]').first();
    const pwdFormExists = await currentPwdInput.isVisible().catch(() => false);
    console.log(`  Password form visible: ${pwdFormExists}`);
    
    // Check for success message element
    const successMsgSelector = '.fixed .text-running, .fixed [class*="success"]';
    console.log(`  Success message area exists: Will test after actual change`);
    
    results.push({ test: 'Password Form', passed: pwdFormExists });
    console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ FAILED');

    // ═══════════════════════════════════════════════════════════
    // TEST 5: NOTIFICATION CHANNELS
    // ═══════════════════════════════════════════════════════════
    console.log('\n' + '═'.repeat(60));
    console.log('TEST 5: NOTIFICATION CHANNELS & SETTINGS');
    console.log('═'.repeat(60));
    
    // Go back
    const backBtn2 = page.locator('.fixed button:has-text("Back"), .fixed button:has-text("Volver")').first();
    if (await backBtn2.isVisible()) {
      await backBtn2.click();
      await page.waitForTimeout(500);
    }
    
    // Click Notifications
    const notifyBtn = page.locator('button:has-text("Notifications"), button:has-text("Notificaciones")').first();
    if (await notifyBtn.isVisible()) {
      await notifyBtn.click();
      await page.waitForTimeout(800);
    }
    
    await page.screenshot({ path: 'test-screenshots/comp-05-notifications.png', fullPage: true });
    
    // Check for notification toggles
    const emailToggle = page.locator('label:has(input[type="checkbox"]):near(:text("Email"))').first();
    const telegramToggle = page.locator('label:has(input[type="checkbox"]):near(:text("Telegram"))').first();
    
    const emailToggleExists = await emailToggle.isVisible().catch(() => false);
    const telegramToggleExists = await telegramToggle.isVisible().catch(() => false);
    
    console.log(`  Email toggle: ${emailToggleExists}`);
    console.log(`  Telegram toggle: ${telegramToggleExists}`);
    
    // Check for test button
    const testNotifyBtn = page.locator('button:has-text("Test"), button:has-text("Probar")').first();
    const testBtnExists = await testNotifyBtn.isVisible().catch(() => false);
    console.log(`  Test notification button: ${testBtnExists}`);
    
    // Check for channel selection
    const channelBtns = page.locator('button:has-text("Telegram"), button:has-text("Email"), button:has-text("Both"), button:has-text("Ambos")');
    const channelCount = await channelBtns.count();
    console.log(`  Channel selection buttons: ${channelCount}`);
    
    results.push({ test: 'Notification Channels', passed: testBtnExists && channelCount >= 2 });
    console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ FAILED');

    // ═══════════════════════════════════════════════════════════
    // TEST 6: USERS MANAGEMENT - DELETE USER
    // ═══════════════════════════════════════════════════════════
    console.log('\n' + '═'.repeat(60));
    console.log('TEST 6: USER MANAGEMENT - DELETE USER');
    console.log('═'.repeat(60));
    
    // Go back
    const backBtn3 = page.locator('.fixed button:has-text("Back"), .fixed button:has-text("Volver")').first();
    if (await backBtn3.isVisible()) {
      await backBtn3.click();
      await page.waitForTimeout(500);
    }
    
    // Click Users
    const usersBtn = page.locator('button:has-text("Users"), button:has-text("Usuarios")').first();
    if (await usersBtn.isVisible()) {
      await usersBtn.click();
      await page.waitForTimeout(1000);
    }
    
    await page.screenshot({ path: 'test-screenshots/comp-06-users.png', fullPage: true });
    
    // Count delete buttons (trash icons)
    const deleteButtons = page.locator('.fixed button:has(svg[class*="Trash2"]), .fixed button svg.lucide-trash-2').locator('..');
    const deleteCount = await deleteButtons.count();
    console.log(`  Delete buttons found: ${deleteCount}`);
    
    // Check if users are displayed
    const userCards = page.locator('.fixed [class*="rounded-lg"]:has(.text-primary):has-text("@")');
    const userCount = await userCards.count().catch(() => 0);
    console.log(`  User cards displayed: ${userCount}`);
    
    results.push({ test: 'Users Management', passed: deleteCount > 0 || userCount > 0 });
    console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ NEEDS INVESTIGATION');

    // ═══════════════════════════════════════════════════════════
    // TEST 7: FORGOT PASSWORD FLOW
    // ═══════════════════════════════════════════════════════════
    console.log('\n' + '═'.repeat(60));
    console.log('TEST 7: FORGOT PASSWORD FLOW');
    console.log('═'.repeat(60));
    
    // Close settings modal first
    await page.click('.fixed button:has(svg[class*="X"])');
    await page.waitForTimeout(500);
    
    // Logout first
    await page.locator('.user-menu-container button').first().click();
    await page.waitForTimeout(300);
    const logoutBtn = page.locator('button:has-text("Sign Out"), button:has-text("Cerrar Sesión")').first();
    await logoutBtn.click();
    await page.waitForTimeout(1500);
    
    await page.screenshot({ path: 'test-screenshots/comp-07-login-for-forgot.png', fullPage: true });
    
    // Click Forgot Password link
    const forgotLink = page.locator('button:has-text("Forgot"), button:has-text("Olvidaste")').first();
    const forgotLinkExists = await forgotLink.isVisible().catch(() => false);
    console.log(`  Forgot password link visible: ${forgotLinkExists}`);
    
    if (forgotLinkExists) {
      await forgotLink.click();
      await page.waitForTimeout(800);
      
      await page.screenshot({ path: 'test-screenshots/comp-07-forgot-form.png', fullPage: true });
      
      // Check for username input
      const usernameInput = page.locator('input[placeholder*="user"], input[type="text"]').first();
      const hasUsernameInput = await usernameInput.isVisible().catch(() => false);
      console.log(`  Username input visible: ${hasUsernameInput}`);
      
      // Check for Send Code button
      const sendCodeBtn = page.locator('button:has-text("Send"), button:has-text("Enviar")').first();
      const hasSendBtn = await sendCodeBtn.isVisible().catch(() => false);
      console.log(`  Send code button visible: ${hasSendBtn}`);
      
      // Test the forgot password API directly
      if (hasUsernameInput && hasSendBtn) {
        await usernameInput.fill('admin');
        await page.waitForTimeout(300);
        
        // Listen for network response
        const responsePromise = page.waitForResponse(resp => 
          resp.url().includes('/auth/forgot-password') && resp.status() === 200
        ).catch(() => null);
        
        await sendCodeBtn.click();
        await page.waitForTimeout(2000);
        
        const response = await responsePromise;
        if (response) {
          const data = await response.json().catch(() => ({}));
          console.log(`  API Response: ${JSON.stringify(data)}`);
          
          // Check if we moved to verify step
          const verifyView = page.locator('input[placeholder*="6"], input[maxlength="6"], :text("Code sent"), :text("Código enviado")').first();
          const onVerifyStep = await verifyView.isVisible().catch(() => false);
          console.log(`  Moved to verify step: ${onVerifyStep}`);
          
          await page.screenshot({ path: 'test-screenshots/comp-07-forgot-result.png', fullPage: true });
        } else {
          console.log('  ⚠️ No API response captured');
        }
      }
    }
    
    results.push({ test: 'Forgot Password', passed: forgotLinkExists });
    console.log(results[results.length-1].passed ? '  ✅ LINK EXISTS' : '  ❌ FAILED');

    // ═══════════════════════════════════════════════════════════
    // TEST 8: TEST TELEGRAM NOTIFICATION (API Test)
    // ═══════════════════════════════════════════════════════════
    console.log('\n' + '═'.repeat(60));
    console.log('TEST 8: TELEGRAM NOTIFICATION (API)');
    console.log('═'.repeat(60));
    
    // Login again
    await page.goto(BASE_URL, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await page.fill('input[type="text"], input[placeholder*="user"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2500);
    
    // Get fresh token
    authToken = await page.evaluate(() => localStorage.getItem('auth_access_token'));
    
    // Test notification via API
    const notifyResult = await page.evaluate(async (token) => {
      try {
        const res = await fetch('http://192.168.1.145:3001/api/notify/test', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
          },
          body: JSON.stringify({
            title: '🧪 Test from Playwright',
            body: 'This is a comprehensive test notification.',
            type: 'info',
            channel: 'telegram'
          })
        });
        return await res.json();
      } catch (e) {
        return { error: e.message };
      }
    }, authToken);
    
    console.log(`  Telegram notification result: ${JSON.stringify(notifyResult)}`);
    results.push({ test: 'Telegram API', passed: notifyResult.success || notifyResult.telegram === 'sent' });
    console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ FAILED');

    // ═══════════════════════════════════════════════════════════
    // TEST 9: TEST EMAIL NOTIFICATION (API)
    // ═══════════════════════════════════════════════════════════
    console.log('\n' + '═'.repeat(60));
    console.log('TEST 9: EMAIL NOTIFICATION (API)');
    console.log('═'.repeat(60));
    
    const emailResult = await page.evaluate(async (token) => {
      try {
        const res = await fetch('http://192.168.1.145:3001/api/notify/test', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
          },
          body: JSON.stringify({
            title: '🧪 Test from Playwright',
            body: 'This is a comprehensive test email notification.',
            type: 'info',
            channel: 'email'
          })
        });
        return await res.json();
      } catch (e) {
        return { error: e.message };
      }
    }, authToken);
    
    console.log(`  Email notification result: ${JSON.stringify(emailResult)}`);
    results.push({ test: 'Email API', passed: emailResult.success || emailResult.email === 'sent' });
    console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ FAILED');

  } catch (error) {
    console.error('\n💥 TEST ERROR:', error);
    await page.screenshot({ path: 'test-screenshots/comp-error.png', fullPage: true });
  } finally {
    // ═══════════════════════════════════════════════════════════
    // SUMMARY
    // ═══════════════════════════════════════════════════════════
    console.log('\n' + '═'.repeat(60));
    console.log('📊 TEST SUMMARY');
    console.log('═'.repeat(60));
    
    const passed = results.filter(r => r.passed).length;
    const failed = results.filter(r => !r.passed).length;
    
    results.forEach(r => {
      console.log(`  ${r.passed ? '✅' : '❌'} ${r.test}`);
    });
    
    console.log('\n' + '─'.repeat(60));
    console.log(`  Total: ${results.length} | Passed: ${passed} | Failed: ${failed}`);
    console.log('═'.repeat(60));
    
    console.log('\n⏳ Keeping browser open for 10 seconds for inspection...');
    await page.waitForTimeout(10000);
    
    await browser.close();
  }
}

comprehensiveTest().catch(console.error);
