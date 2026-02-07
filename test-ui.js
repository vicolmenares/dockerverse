const { chromium } = require('playwright');

const BASE_URL = 'http://192.168.1.145:3002';
const CREDENTIALS = { username: 'admin', password: 'admin123' };

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function runTests() {
  console.log('🚀 Starting DockerVerse UI Tests...\n');
  
  const browser = await chromium.launch({ 
    headless: false, // Visible browser for debugging
    slowMo: 500 // Slow down actions
  });
  
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 }
  });
  
  const page = await context.newPage();
  
  // Capture console logs
  page.on('console', msg => {
    if (msg.type() === 'error') {
      console.log(`  ❌ Console Error: ${msg.text()}`);
    }
  });

  let testsPassed = 0;
  let testsFailed = 0;

  try {
    // ===========================================
    // TEST 1: Login
    // ===========================================
    console.log('📌 TEST 1: Login');
    await page.goto(BASE_URL);
    await sleep(1000);
    
    // Should show login page
    const loginForm = await page.locator('input[type="text"], input[name="username"]').first();
    if (await loginForm.isVisible()) {
      console.log('  ✅ Login page displayed');
      
      // Fill credentials
      await page.fill('input[type="text"], input[name="username"]', CREDENTIALS.username);
      await page.fill('input[type="password"]', CREDENTIALS.password);
      await page.click('button[type="submit"]');
      await sleep(2000);
      
      // Check if logged in (should see dashboard/containers)
      const loggedIn = await page.locator('text=Containers, text=Dashboard, text=Contenedores').first().isVisible().catch(() => false);
      if (loggedIn) {
        console.log('  ✅ Login successful\n');
        testsPassed++;
      } else {
        console.log('  ❌ Login failed - dashboard not visible\n');
        testsFailed++;
      }
    } else {
      console.log('  ❌ Login form not found\n');
      testsFailed++;
    }

    // Take screenshot after login
    await page.screenshot({ path: 'test-screenshots/01-after-login.png', fullPage: true });

    // ===========================================
    // TEST 2: Session Persistence (Page Refresh)
    // ===========================================
    console.log('📌 TEST 2: Session Persistence (Page Refresh)');
    await page.reload();
    await sleep(2000);
    
    // Check if still logged in
    const stillLoggedIn = await page.locator('input[type="password"]').isVisible().catch(() => false);
    if (!stillLoggedIn) {
      console.log('  ✅ Session persisted after refresh\n');
      testsPassed++;
    } else {
      console.log('  ❌ Session lost after refresh - back to login\n');
      testsFailed++;
    }
    
    await page.screenshot({ path: 'test-screenshots/02-after-refresh.png', fullPage: true });

    // ===========================================
    // TEST 3: User Menu in Header
    // ===========================================
    console.log('📌 TEST 3: User Menu in Header');
    
    // Look for user avatar/icon in header
    const userMenuBtn = await page.locator('button:has([class*="user"]), button:has(svg), .relative button').last();
    if (await userMenuBtn.isVisible()) {
      await userMenuBtn.click();
      await sleep(500);
      
      // Check if dropdown appears with settings/logout options
      const dropdownVisible = await page.locator('text=Settings, text=Logout, text=Configuración, text=Cerrar sesión').first().isVisible().catch(() => false);
      if (dropdownVisible) {
        console.log('  ✅ User menu dropdown works\n');
        testsPassed++;
      } else {
        console.log('  ⚠️ Menu clicked but dropdown not detected\n');
        testsFailed++;
      }
      
      await page.screenshot({ path: 'test-screenshots/03-user-menu.png', fullPage: true });
      
      // Close menu by clicking elsewhere
      await page.click('body', { position: { x: 100, y: 100 } });
    } else {
      console.log('  ❌ User menu button not found in header\n');
      testsFailed++;
    }

    // ===========================================
    // TEST 4: Language Switch (Spanish)
    // ===========================================
    console.log('📌 TEST 4: Language Switch');
    
    // Open settings
    const settingsBtn = await page.locator('button:has-text("Settings"), button:has-text("Configuración"), [aria-label*="settings"]').first();
    if (await settingsBtn.isVisible().catch(() => false)) {
      await settingsBtn.click();
      await sleep(1000);
    } else {
      // Try opening via user menu
      const userBtn = await page.locator('button:has(svg)').last();
      await userBtn.click();
      await sleep(500);
      await page.click('text=Settings, text=Configuración');
      await sleep(1000);
    }
    
    // Find language selector
    const langSelector = await page.locator('select, [role="combobox"]').first();
    if (await langSelector.isVisible().catch(() => false)) {
      await langSelector.selectOption('es');
      await sleep(500);
      
      // Check for Spanish text
      const spanishText = await page.locator('text=Usuarios, text=Notificaciones, text=Apariencia').first().isVisible().catch(() => false);
      if (spanishText) {
        console.log('  ✅ Language switched to Spanish\n');
        testsPassed++;
      } else {
        console.log('  ⚠️ Language selector found but Spanish text not detected\n');
        testsFailed++;
      }
    } else {
      console.log('  ⚠️ Language selector not found in settings\n');
      testsFailed++;
    }
    
    await page.screenshot({ path: 'test-screenshots/04-spanish-settings.png', fullPage: true });

    // ===========================================
    // TEST 5: Unified Notifications Section
    // ===========================================
    console.log('📌 TEST 5: Notifications Section');
    
    // Click on Notifications/Notificaciones
    const notifTab = await page.locator('text=Notifications, text=Notificaciones').first();
    if (await notifTab.isVisible()) {
      await notifTab.click();
      await sleep(500);
      
      // Check for unified elements (thresholds, apprise config)
      const hasThresholds = await page.locator('text=CPU, text=Memory, text=Threshold').first().isVisible().catch(() => false);
      const hasApprise = await page.locator('text=Apprise, text=apprise').first().isVisible().catch(() => false);
      
      if (hasThresholds && hasApprise) {
        console.log('  ✅ Notifications section is unified (has thresholds + Apprise)\n');
        testsPassed++;
      } else {
        console.log(`  ⚠️ Notifications partial: Thresholds=${hasThresholds}, Apprise=${hasApprise}\n`);
        testsFailed++;
      }
    } else {
      console.log('  ❌ Notifications tab not found\n');
      testsFailed++;
    }
    
    await page.screenshot({ path: 'test-screenshots/05-notifications.png', fullPage: true });

    // ===========================================
    // TEST 6: Users Section & Create User
    // ===========================================
    console.log('📌 TEST 6: Users Section & Create User');
    
    // Click on Users/Usuarios
    const usersTab = await page.locator('text=Users, text=Usuarios').first();
    if (await usersTab.isVisible()) {
      await usersTab.click();
      await sleep(500);
      
      await page.screenshot({ path: 'test-screenshots/06a-users-list.png', fullPage: true });
      
      // Check for Add button
      const addBtn = await page.locator('button:has-text("Add"), button:has-text("Agregar"), button:has(svg)').first();
      if (await addBtn.isVisible()) {
        console.log('  ✅ Users section loads correctly\n');
        testsPassed++;
        
        // Try to create a user
        await addBtn.click();
        await sleep(500);
        
        // Fill user form
        const usernameInput = await page.locator('input[name="username"], input[placeholder*="username"]').first();
        if (await usernameInput.isVisible()) {
          await usernameInput.fill('testuser123');
          
          const emailInput = await page.locator('input[name="email"], input[type="email"]').first();
          if (await emailInput.isVisible()) await emailInput.fill('test@test.com');
          
          const passInput = await page.locator('input[name="password"], input[type="password"]').first();
          if (await passInput.isVisible()) await passInput.fill('password123');
          
          await page.screenshot({ path: 'test-screenshots/06b-user-form.png', fullPage: true });
          
          // Click Save
          const saveBtn = await page.locator('button:has-text("Save"), button:has-text("Guardar")').first();
          if (await saveBtn.isVisible()) {
            await saveBtn.click();
            await sleep(2000);
            
            // Check for success or error
            const error = await page.locator('text=error, text=Error, .text-red').first().isVisible().catch(() => false);
            if (!error) {
              console.log('  ✅ User creation - Save button worked\n');
              testsPassed++;
            } else {
              console.log('  ❌ User creation - Error after save\n');
              testsFailed++;
            }
          }
        }
      }
    } else {
      console.log('  ❌ Users tab not found\n');
      testsFailed++;
    }
    
    await page.screenshot({ path: 'test-screenshots/06c-after-save.png', fullPage: true });

  } catch (error) {
    console.log(`\n❌ Test Error: ${error.message}`);
    await page.screenshot({ path: 'test-screenshots/error.png', fullPage: true });
  }

  // ===========================================
  // RESULTS
  // ===========================================
  console.log('\n========================================');
  console.log(`📊 TEST RESULTS: ${testsPassed} passed, ${testsFailed} failed`);
  console.log('========================================\n');

  await browser.close();
}

// Create screenshots folder
const fs = require('fs');
if (!fs.existsSync('test-screenshots')) {
  fs.mkdirSync('test-screenshots');
}

runTests().catch(console.error);
