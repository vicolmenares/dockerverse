const { chromium } = require('playwright');

const BASE_URL = 'http://192.168.1.145:3002';
const API_URL = 'http://192.168.1.145:3001';

async function debugForgotPassword() {
  console.log('🔍 DEBUG: Forgot Password Flow (v2)\n');
  
  const browser = await chromium.launch({ headless: false, slowMo: 400 });
  const page = await browser.newPage({ viewport: { width: 1200, height: 800 } });

  // Capture console logs
  page.on('console', msg => {
    if (msg.type() === 'error') {
      console.log(`  [Browser Error]: ${msg.text().substring(0, 200)}`);
    }
  });

  // Capture network requests
  page.on('request', req => {
    if (req.url().includes('forgot-password')) {
      console.log(`  [REQUEST] ${req.method()} ${req.url()}`);
      console.log(`  [REQUEST BODY] ${req.postData()}`);
    }
  });
  
  page.on('response', async res => {
    if (res.url().includes('forgot-password')) {
      console.log(`  [RESPONSE] ${res.status()}`);
      try {
        const body = await res.json();
        console.log(`  [RESPONSE BODY] ${JSON.stringify(body)}`);
      } catch (e) {}
    }
  });

  try {
    console.log('1. Navigating to login page...');
    await page.goto(BASE_URL, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    
    console.log('\n2. Looking for Forgot Password link...');
    const forgotLink = page.locator('button:has-text("Forgot"), button:has-text("Olvidaste")').first();
    const foundLink = await forgotLink.isVisible();
    console.log(`   Found: ${foundLink}`);
    
    if (foundLink) {
      console.log('\n3. Clicking Forgot Password...');
      await forgotLink.click();
      await page.waitForTimeout(1000);
      
      // List all buttons
      console.log('\n4. Listing all buttons on page:');
      const allButtons = page.locator('button');
      const buttonCount = await allButtons.count();
      for (let i = 0; i < buttonCount; i++) {
        const txt = await allButtons.nth(i).textContent().catch(() => '');
        console.log(`   Button ${i}: "${txt.trim()}"`);
      }
      
      // Try different selectors for the send button
      console.log('\n5. Trying different selectors:');
      const selectors = [
        'button:has-text("Send Code")',
        'button:has-text("Enviar Código")',
        'button:has-text("Code")',
        'button:has-text("Código")',
        'button[class*="primary"]:not(:disabled)',
        'button.bg-primary:not(:disabled)',
      ];
      
      for (const sel of selectors) {
        const btn = page.locator(sel).first();
        const visible = await btn.isVisible().catch(() => false);
        console.log(`   ${sel}: ${visible ? 'FOUND' : 'not found'}`);
      }
      
      // Fill username
      console.log('\n6. Filling username...');
      const usernameInput = page.locator('input[type="text"]').first();
      await usernameInput.fill('admin');
      await page.waitForTimeout(500);
      
      // Try to find the enabled primary button
      console.log('\n7. Finding enabled submit button...');
      const submitBtn = page.locator('button.bg-primary:not([disabled])').first();
      const submitExists = await submitBtn.isVisible();
      console.log(`   Submit button visible: ${submitExists}`);
      
      if (submitExists) {
        const btnText = await submitBtn.textContent();
        console.log(`   Button text: "${btnText.trim()}"`);
        
        console.log('\n8. Clicking submit button...');
        await submitBtn.click();
        await page.waitForTimeout(3000);
        
        await page.screenshot({ path: 'test-screenshots/debug-forgot-v2-result.png' });
        
        // Check what happened
        const errorEl = page.locator('.text-stopped');
        const hasError = await errorEl.isVisible().catch(() => false);
        if (hasError) {
          const errorText = await errorEl.textContent();
          console.log(`   ❌ ERROR: ${errorText}`);
        }
        
        const codeInput = page.locator('input[maxlength="6"]');
        const onVerify = await codeInput.isVisible().catch(() => false);
        console.log(`   On verify step: ${onVerify}`);
        
        if (onVerify) {
          console.log('   ✅ Successfully moved to verify step!');
          const maskedEmail = await page.locator('.font-mono').textContent().catch(() => '');
          console.log(`   Masked email: ${maskedEmail}`);
        }
      }
    }

  } catch (error) {
    console.error('\n💥 ERROR:', error.message);
    await page.screenshot({ path: 'test-screenshots/debug-forgot-v2-error.png' });
  } finally {
    console.log('\n⏳ Browser open for 10s...');
    await page.waitForTimeout(10000);
    await browser.close();
  }
}

debugForgotPassword().catch(console.error);
