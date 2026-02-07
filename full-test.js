const { chromium } = require('playwright');

const BASE_URL = 'http://192.168.1.145:3002';

async function fullTest() {
  console.log('🧪 Full DockerVerse UI Test\n');
  
  const browser = await chromium.launch({ headless: false, slowMo: 800 });
  const page = await browser.newPage();
  
  page.on('console', msg => {
    if (msg.type() === 'error') console.log(`[Error]: ${msg.text()}`);
  });

  const results = [];
  
  try {
    // LOGIN
    console.log('1️⃣ LOGIN');
    await page.goto(BASE_URL);
    await page.waitForTimeout(1000);
    await page.fill('input[placeholder="username"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    
    const loggedIn = await page.locator('text=Containers').first().isVisible();
    results.push({ test: 'Login', passed: loggedIn });
    console.log(loggedIn ? '  ✅ Login OK' : '  ❌ Login FAILED');
    
    // SESSION PERSISTENCE
    console.log('2️⃣ SESSION PERSISTENCE');
    await page.reload();
    await page.waitForTimeout(2000);
    const stillLoggedIn = !(await page.locator('input[type="password"]').isVisible());
    results.push({ test: 'Session Persistence', passed: stillLoggedIn });
    console.log(stillLoggedIn ? '  ✅ Session persists after refresh' : '  ❌ Session LOST');
    
    // USER MENU
    console.log('3️⃣ USER MENU IN HEADER');
    await page.screenshot({ path: 'test-screenshots/full-01-dashboard.png', fullPage: true });
    
    // Find user menu button (last relative button or button with user icon)
    const userMenuBtn = page.locator('header .relative button, button:has(.w-8.h-8)').first();
    const menuVisible = await userMenuBtn.isVisible();
    console.log(`  User menu button found: ${menuVisible}`);
    
    if (menuVisible) {
      await userMenuBtn.click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: 'test-screenshots/full-02-user-menu.png', fullPage: true });
      
      // Check for dropdown content
      const dropdownItems = await page.locator('.absolute button, .absolute a').count();
      console.log(`  Dropdown items found: ${dropdownItems}`);
      
      const hasSettings = await page.locator('text=Settings, text=Configuración').first().isVisible().catch(() => false);
      const hasLogout = await page.locator('text=Logout, text=Cerrar, text=Log out').first().isVisible().catch(() => false);
      
      results.push({ test: 'User Menu Dropdown', passed: dropdownItems > 0 || hasSettings || hasLogout });
      console.log(`  Settings visible: ${hasSettings}, Logout visible: ${hasLogout}`);
      
      // Click Settings
      if (hasSettings) {
        await page.click('text=Settings, text=Configuración');
        await page.waitForTimeout(1000);
      }
    } else {
      results.push({ test: 'User Menu Dropdown', passed: false });
    }
    
    // SETTINGS MODAL
    console.log('4️⃣ SETTINGS MODAL');
    // If not open yet, try keyboard shortcut or button
    let settingsOpen = await page.locator('[role="dialog"], .fixed.inset-0, .modal').first().isVisible().catch(() => false);
    
    if (!settingsOpen) {
      // Try gear icon button
      await page.click('button:has(svg)').catch(() => {});
      await page.waitForTimeout(500);
      settingsOpen = await page.locator('[role="dialog"], .fixed.inset-0').first().isVisible().catch(() => false);
    }
    
    await page.screenshot({ path: 'test-screenshots/full-03-settings.png', fullPage: true });
    results.push({ test: 'Settings Modal', passed: settingsOpen });
    console.log(settingsOpen ? '  ✅ Settings modal opens' : '  ❌ Settings modal NOT found');
    
    if (settingsOpen) {
      // LANGUAGE SWITCH
      console.log('5️⃣ LANGUAGE SWITCH');
      const langBtn = page.locator('button:has-text("EN"), button:has-text("ES"), select').first();
      const langVisible = await langBtn.isVisible();
      console.log(`  Language control found: ${langVisible}`);
      
      // Get current language items
      const menuItems = await page.locator('nav button, aside button, [role="menuitem"]').allTextContents();
      console.log(`  Menu items: ${menuItems.join(', ')}`);
      
      // Try to find and click ES
      const esOption = page.locator('button:has-text("ES")').first();
      if (await esOption.isVisible()) {
        await esOption.click();
        await page.waitForTimeout(500);
        
        // Check for Spanish text
        const spanishFound = await page.locator('text=Usuarios, text=Notificaciones, text=Idioma').first().isVisible().catch(() => false);
        results.push({ test: 'Language Spanish', passed: spanishFound });
        console.log(spanishFound ? '  ✅ Spanish translation works' : '  ⚠️ Spanish text not detected');
      } else {
        results.push({ test: 'Language Spanish', passed: false });
      }
      
      await page.screenshot({ path: 'test-screenshots/full-04-spanish.png', fullPage: true });
      
      // NOTIFICATIONS
      console.log('6️⃣ NOTIFICATIONS SECTION');
      const notifBtn = page.locator('button:has-text("Notificaciones"), button:has-text("Notifications")').first();
      if (await notifBtn.isVisible()) {
        await notifBtn.click();
        await page.waitForTimeout(500);
        
        await page.screenshot({ path: 'test-screenshots/full-05-notifications.png', fullPage: true });
        
        // Check for unified content
        const hasThresholds = await page.locator('text=CPU, text=Memory, input[type="range"], input[type="number"]').first().isVisible().catch(() => false);
        const hasApprise = await page.locator('text=Apprise, input[placeholder*="apprise"]').first().isVisible().catch(() => false);
        
        results.push({ test: 'Notifications Unified', passed: hasThresholds });
        console.log(`  Thresholds: ${hasThresholds}, Apprise: ${hasApprise}`);
      }
      
      // USERS
      console.log('7️⃣ USERS SECTION');
      const usersBtn = page.locator('button:has-text("Usuarios"), button:has-text("Users")').first();
      if (await usersBtn.isVisible()) {
        await usersBtn.click();
        await page.waitForTimeout(500);
        
        await page.screenshot({ path: 'test-screenshots/full-06-users.png', fullPage: true });
        
        // Check for user list
        const hasUserList = await page.locator('text=admin').isVisible().catch(() => false);
        console.log(`  User list visible: ${hasUserList}`);
        
        // Try to add user
        const addBtn = page.locator('button:has(svg[class*="plus"]), button:has-text("+"), button:has-text("Add")').first();
        if (await addBtn.isVisible()) {
          await addBtn.click();
          await page.waitForTimeout(500);
          
          await page.screenshot({ path: 'test-screenshots/full-07-add-user.png', fullPage: true });
          
          // Fill form
          const usernameInput = page.locator('input[placeholder*="username"], input[name="username"]').first();
          if (await usernameInput.isVisible()) {
            await usernameInput.fill('uitestuser');
            
            const emailInput = page.locator('input[type="email"], input[placeholder*="email"]').first();
            if (await emailInput.isVisible()) await emailInput.fill('test@test.com');
            
            const passInput = page.locator('input[type="password"]');
            if (await passInput.count() > 0) await passInput.first().fill('testpass123');
            
            await page.screenshot({ path: 'test-screenshots/full-08-user-form.png', fullPage: true });
            
            // Click Save
            const saveBtn = page.locator('button:has-text("Save"), button:has-text("Guardar")').first();
            if (await saveBtn.isVisible()) {
              console.log('  Clicking Save...');
              await saveBtn.click();
              await page.waitForTimeout(2000);
              
              await page.screenshot({ path: 'test-screenshots/full-09-after-save.png', fullPage: true });
              
              // Check for new user in list or success message
              const userCreated = await page.locator('text=uitestuser').isVisible().catch(() => false);
              results.push({ test: 'User Creation', passed: userCreated });
              console.log(userCreated ? '  ✅ User created successfully' : '  ⚠️ User might not have been created');
            }
          }
        }
      }
    }
    
  } catch (error) {
    console.log(`\n❌ Test Error: ${error.message}`);
    await page.screenshot({ path: 'test-screenshots/full-error.png', fullPage: true });
  }

  // RESULTS
  console.log('\n========================================');
  console.log('📊 FINAL TEST RESULTS');
  console.log('========================================');
  
  let passed = 0, failed = 0;
  for (const r of results) {
    console.log(`${r.passed ? '✅' : '❌'} ${r.test}`);
    if (r.passed) passed++; else failed++;
  }
  
  console.log(`\nTotal: ${passed}/${results.length} passed`);
  console.log('========================================\n');
  
  console.log('Screenshots saved in test-screenshots/');
  console.log('Browser closing in 10 seconds...');
  await page.waitForTimeout(10000);
  
  await browser.close();
}

fullTest().catch(console.error);
