const { chromium } = require('playwright');

const BASE_URL = 'http://192.168.1.145:3002';

async function finalTest() {
  console.log('🎯 Final Comprehensive DockerVerse UI Test\n');
  
  const browser = await chromium.launch({ headless: false, slowMo: 600 });
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });

  const results = [];

  try {
    // 1. LOGIN
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('TEST 1: LOGIN');
    await page.goto(BASE_URL);
    await page.waitForTimeout(1500);
    await page.fill('input[placeholder="username"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(3000);
    
    const loggedIn = await page.locator('h1:has-text("DockerVerse")').isVisible();
    const containersVisible = await page.locator('text=Containers').first().isVisible().catch(() => false);
    const isLoginPage = await page.locator('input[type="password"]').isVisible();
    
    results.push({ test: 'Login', passed: loggedIn && !isLoginPage });
    console.log(`  Dashboard visible: ${loggedIn}, Containers: ${containersVisible}, Still on login: ${isLoginPage}`);
    console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ FAILED');
    
    await page.screenshot({ path: 'test-screenshots/final-01-login.png', fullPage: true });

    // 2. SESSION PERSISTENCE
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('TEST 2: SESSION PERSISTENCE (Refresh)');
    await page.reload({ waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    
    const stillOnDashboard = await page.locator('h1:has-text("DockerVerse")').isVisible();
    const backToLogin = await page.locator('input[type="password"]').isVisible();
    
    results.push({ test: 'Session Persistence', passed: stillOnDashboard && !backToLogin });
    console.log(`  Still on dashboard: ${stillOnDashboard}, Back to login: ${backToLogin}`);
    console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ FAILED');

    await page.screenshot({ path: 'test-screenshots/final-02-refresh.png', fullPage: true });

    // 3. USER MENU
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('TEST 3: USER MENU IN HEADER');
    
    // Find the user menu container
    const userMenuBtn = page.locator('.user-menu-container button').first();
    const menuExists = await userMenuBtn.isVisible();
    console.log(`  User menu button found: ${menuExists}`);
    
    if (menuExists) {
      await userMenuBtn.click();
      await page.waitForTimeout(500);
      
      await page.screenshot({ path: 'test-screenshots/final-03-user-menu-open.png', fullPage: true });
      
      // Check dropdown
      const dropdown = page.locator('.user-menu-container .absolute');
      const dropdownVisible = await dropdown.isVisible();
      console.log(`  Dropdown visible: ${dropdownVisible}`);
      
      // Check for menu items
      const settingsBtn = page.locator('.user-menu-container button:has-text("Settings"), .user-menu-container button:has-text("Configuración")').first();
      const logoutBtn = page.locator('.user-menu-container button:has-text("Sign Out"), .user-menu-container button:has-text("Cerrar")').first();
      
      const hasSettings = await settingsBtn.isVisible().catch(() => false);
      const hasLogout = await logoutBtn.isVisible().catch(() => false);
      
      console.log(`  Settings option: ${hasSettings}, Logout option: ${hasLogout}`);
      results.push({ test: 'User Menu', passed: dropdownVisible && (hasSettings || hasLogout) });
      console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ FAILED');
      
      // Click Settings to open modal
      if (hasSettings) {
        await settingsBtn.click();
        await page.waitForTimeout(800);
      }
    } else {
      results.push({ test: 'User Menu', passed: false });
      console.log('  ❌ FAILED - User menu not found');
    }

    // 4. SETTINGS MODAL
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('TEST 4: SETTINGS MODAL');
    
    await page.screenshot({ path: 'test-screenshots/final-04-settings-modal.png', fullPage: true });
    
    // Check for settings modal
    const settingsModal = page.locator('.fixed.inset-0, [role="dialog"]').first();
    const modalOpen = await settingsModal.isVisible();
    console.log(`  Settings modal open: ${modalOpen}`);
    
    // Find menu items in the modal
    const allButtons = await page.locator('.bg-background-secondary button').allTextContents();
    console.log(`  Modal buttons: ${allButtons.slice(0, 5).join(', ')}...`);
    
    // Look for key items
    const hasUsersMenu = allButtons.some(t => t.includes('Users') || t.includes('Usuarios'));
    const hasNotifMenu = allButtons.some(t => t.includes('Notification') || t.includes('Notificacion'));
    const hasLangMenu = allButtons.some(t => t.includes('Language') || t.includes('Idioma'));
    
    console.log(`  Users menu: ${hasUsersMenu}, Notifications: ${hasNotifMenu}, Language: ${hasLangMenu}`);
    results.push({ test: 'Settings Modal', passed: modalOpen && (hasUsersMenu || hasNotifMenu) });
    console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ FAILED');

    // 5. LANGUAGE SWITCH (check Spanish)
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('TEST 5: LANGUAGE SWITCH');
    
    // Find Language/Idioma option and click it
    const langOption = page.locator('button:has-text("Language"), button:has-text("Idioma")').first();
    if (await langOption.isVisible()) {
      await langOption.click();
      await page.waitForTimeout(500);
      
      await page.screenshot({ path: 'test-screenshots/final-05-language-panel.png', fullPage: true });
      
      // Look for ES option
      const esBtn = page.locator('button:has-text("Español")').first();
      if (await esBtn.isVisible().catch(() => false)) {
        await esBtn.click();
        await page.waitForTimeout(500);
      }
      
      // Go back to main menu
      const backBtn = page.locator('button:has-text("Back"), button:has-text("Volver")').first();
      if (await backBtn.isVisible()) {
        await backBtn.click();
        await page.waitForTimeout(500);
      }
      
      await page.screenshot({ path: 'test-screenshots/final-06-spanish.png', fullPage: true });
      
      // Check if we now see Spanish text
      const spanishButtons = await page.locator('.bg-background-secondary button').allTextContents();
      const hasSpanish = spanishButtons.some(t => 
        t.includes('Usuarios') || t.includes('Notificaciones') || t.includes('Perfil')
      );
      
      console.log(`  Spanish text found: ${hasSpanish}`);
      console.log(`  Menu now shows: ${spanishButtons.slice(0, 4).join(', ')}...`);
      results.push({ test: 'Language Switch', passed: hasSpanish });
      console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ FAILED');
    } else {
      results.push({ test: 'Language Switch', passed: false });
      console.log('  ❌ FAILED - Language option not found');
    }

    // 6. NOTIFICATIONS SECTION
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('TEST 6: NOTIFICATIONS SECTION');
    
    const notifBtn = page.locator('button:has-text("Notificaciones"), button:has-text("Notifications")').first();
    if (await notifBtn.isVisible()) {
      await notifBtn.click();
      await page.waitForTimeout(500);
      
      await page.screenshot({ path: 'test-screenshots/final-07-notifications.png', fullPage: true });
      
      // Check for unified content (thresholds + apprise)
      const pageText = await page.locator('.bg-background-secondary').textContent();
      const hasThresholds = pageText.includes('CPU') || pageText.includes('Memory') || pageText.includes('Memoria');
      const hasApprise = pageText.includes('Apprise') || pageText.includes('apprise');
      const hasToggles = await page.locator('input[type="checkbox"], [role="switch"]').count() > 0;
      
      console.log(`  Has thresholds: ${hasThresholds}, Has Apprise: ${hasApprise}, Has toggles: ${hasToggles}`);
      results.push({ test: 'Notifications Unified', passed: hasThresholds && hasApprise });
      console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ FAILED');
      
      // Go back
      const backBtn = page.locator('button:has-text("Back"), button:has-text("Volver")').first();
      if (await backBtn.isVisible()) await backBtn.click();
      await page.waitForTimeout(300);
    } else {
      results.push({ test: 'Notifications Unified', passed: false });
      console.log('  ❌ FAILED - Notifications option not found');
    }

    // 7. USERS SECTION
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('TEST 7: USERS SECTION');
    
    const usersBtn = page.locator('button:has-text("Usuarios"), button:has-text("Users")').first();
    if (await usersBtn.isVisible()) {
      await usersBtn.click();
      await page.waitForTimeout(500);
      
      await page.screenshot({ path: 'test-screenshots/final-08-users.png', fullPage: true });
      
      // Check for user list
      const hasAdminUser = await page.locator('text=admin').isVisible();
      const hasAddBtn = await page.locator('button:has-text("Agregar"), button:has-text("Add")').isVisible();
      
      console.log(`  Admin user visible: ${hasAdminUser}, Add button: ${hasAddBtn}`);
      results.push({ test: 'Users Section', passed: hasAdminUser });
      console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ❌ FAILED');
      
      // Test create user flow
      if (hasAddBtn) {
        await page.click('button:has-text("Agregar"), button:has-text("Add")');
        await page.waitForTimeout(500);
        
        await page.screenshot({ path: 'test-screenshots/final-09-user-form.png', fullPage: true });
        
        // Fill the form
        await page.fill('input[placeholder*="username"]', 'playwrighttest');
        const emailInput = page.locator('input[type="email"]').first();
        if (await emailInput.isVisible()) await emailInput.fill('playwright@test.com');
        const passInput = page.locator('input[type="password"]').first();
        if (await passInput.isVisible()) await passInput.fill('TestPass123');
        
        await page.screenshot({ path: 'test-screenshots/final-10-user-filled.png', fullPage: true });
        
        // Click Save
        const saveBtn = page.locator('button:has-text("Guardar"), button:has-text("Save")').first();
        console.log(`  Save button visible: ${await saveBtn.isVisible()}`);
        
        if (await saveBtn.isVisible()) {
          await saveBtn.click();
          await page.waitForTimeout(2000);
          
          await page.screenshot({ path: 'test-screenshots/final-11-after-save.png', fullPage: true });
          
          // Check if user was created
          const userCreated = await page.locator('text=playwrighttest').isVisible().catch(() => false);
          console.log(`  User created: ${userCreated}`);
          results.push({ test: 'User Creation', passed: userCreated });
          console.log(results[results.length-1].passed ? '  ✅ PASSED' : '  ⚠️ Check manually');
        }
      }
    } else {
      results.push({ test: 'Users Section', passed: false });
      console.log('  ❌ FAILED - Users option not found (may require admin role)');
    }

  } catch (error) {
    console.log(`\n❌ Test Error: ${error.message}`);
    await page.screenshot({ path: 'test-screenshots/final-error.png', fullPage: true });
  }

  // FINAL RESULTS
  console.log('\n\n╔══════════════════════════════════════════╗');
  console.log('║       FINAL TEST RESULTS SUMMARY         ║');
  console.log('╠══════════════════════════════════════════╣');
  
  let passed = 0, failed = 0;
  for (const r of results) {
    const status = r.passed ? '✅' : '❌';
    const label = r.test.padEnd(25);
    console.log(`║  ${status} ${label} ║`);
    if (r.passed) passed++; else failed++;
  }
  
  console.log('╠══════════════════════════════════════════╣');
  console.log(`║  TOTAL: ${passed}/${results.length} tests passed`.padEnd(43) + '║');
  console.log('╚══════════════════════════════════════════╝');
  
  console.log('\n📸 Screenshots saved in test-screenshots/');
  console.log('🖥️  Browser staying open 15 seconds for inspection...');
  await page.waitForTimeout(15000);
  
  await browser.close();
}

finalTest().catch(console.error);
