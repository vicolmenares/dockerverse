const { chromium } = require('playwright');

const BASE_URL = 'http://192.168.1.145:3002';

async function testUserManagement() {
  console.log('👤 Testing User Create & Delete\n');
  
  const browser = await chromium.launch({ headless: false, slowMo: 250 });
  const context = await browser.newContext({ viewport: { width: 1400, height: 900 } });
  const page = await context.newPage();

  // Track all console logs
  page.on('console', msg => {
    const text = msg.text();
    if (text.includes('user') || text.includes('User') || text.includes('Saving') || text.includes('Error') || text.includes('Response')) {
      console.log(`[Console]: ${text}`);
    }
  });

  // Track API calls
  let apiCalls = [];
  page.on('response', response => {
    const url = response.url();
    if (url.includes('/api/users')) {
      const info = `${response.request().method()} ${url.split('/api')[1]} => ${response.status()}`;
      apiCalls.push(info);
      console.log(`[API]: ${info}`);
    }
  });
  
  page.on('dialog', async dialog => {
    console.log(`[Dialog]: ${dialog.message()}`);
    await dialog.accept();
  });

  try {
    // LOGIN
    console.log('1. Logging in as admin...');
    await page.goto(BASE_URL + '?t=' + Date.now(), { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    await page.fill('#username', 'admin');
    await page.fill('#password', 'admin123');
    await page.click('button:has-text("Sign In")');
    await page.waitForTimeout(3000);
    
    // OPEN SETTINGS > USERS
    console.log('\n2. Opening Settings > Users...');
    await page.locator('button').nth(4).click(); // Admin dropdown
    await page.waitForTimeout(500);
    await page.click('text=Settings');
    await page.waitForTimeout(1500);
    await page.click('text=Users');
    await page.waitForTimeout(1500);
    await page.screenshot({ path: 'test-screenshots/umgmt-1-users.png' });
    
    // Check current users
    const beforeUsers = await page.evaluate(() => {
      const userCards = [...document.querySelectorAll('[class*="bg-background-secondary"]')].filter(el => 
        el.querySelector('svg') && el.textContent.includes('@')
      );
      return userCards.length;
    });
    console.log('   Users before:', beforeUsers);
    
    // CLICK ADD USER
    console.log('\n3. Opening user form...');
    await page.click('text=Add User');
    await page.waitForTimeout(1000);
    await page.screenshot({ path: 'test-screenshots/umgmt-2-form.png' });
    
    // FILL FORM - use keyboard typing for better Svelte compatibility
    console.log('\n4. Filling form...');
    const inputs = page.locator('.grid.grid-cols-2 input');
    const inputCount = await inputs.count();
    console.log('   Found inputs:', inputCount);
    
    // Fill each input by order (username, email, firstName, lastName, password)
    const formData = ['testuser' + Date.now(), 'test@test.com', 'Test', 'User', 'testpass123'];
    for (let i = 0; i < Math.min(inputCount, formData.length); i++) {
      const input = inputs.nth(i);
      await input.click();
      await input.fill('');
      await input.type(formData[i], { delay: 50 });
      console.log(`   Filled input ${i}: ${formData[i]}`);
    }
    
    await page.waitForTimeout(500);
    await page.screenshot({ path: 'test-screenshots/umgmt-3-filled.png' });
    
    // SAVE
    console.log('\n5. Clicking Save...');
    apiCalls = []; // Reset to track save call
    const saveBtn = page.locator('button:has-text("Save"), button:has-text("Guardar")').first();
    await saveBtn.click();
    
    // Wait for API call
    await page.waitForTimeout(3000);
    console.log('   API calls after save:', apiCalls);
    await page.screenshot({ path: 'test-screenshots/umgmt-4-saved.png' });
    
    // Check users after save
    const afterUsers = await page.evaluate(() => {
      const userCards = [...document.querySelectorAll('[class*="bg-background-secondary"]')].filter(el => 
        el.querySelector('svg') && el.textContent.includes('@')
      );
      return userCards.length;
    });
    console.log('   Users after save:', afterUsers);
    
    // Check for trash buttons
    const trashCount = await page.evaluate(() => {
      return [...document.querySelectorAll('button')].filter(b => {
        const svg = b.querySelector('svg');
        return svg && svg.outerHTML.toLowerCase().includes('trash');
      }).length;
    });
    console.log('   Trash buttons:', trashCount);
    
    if (trashCount > 0) {
      // DELETE TEST
      console.log('\n6. Testing delete...');
      apiCalls = [];
      
      // Click the trash button via JS to ensure it works
      await page.evaluate(() => {
        const btns = [...document.querySelectorAll('button')].filter(b => {
          const svg = b.querySelector('svg');
          return svg && svg.outerHTML.toLowerCase().includes('trash');
        });
        if (btns.length > 0) {
          console.log('Clicking delete for user');
          btns[0].click();
        }
      });
      
      await page.waitForTimeout(3000);
      console.log('   API calls after delete:', apiCalls);
      await page.screenshot({ path: 'test-screenshots/umgmt-5-deleted.png' });
      
      // Final check
      const finalTrash = await page.evaluate(() => {
        return [...document.querySelectorAll('button')].filter(b => {
          const svg = b.querySelector('svg');
          return svg && svg.outerHTML.toLowerCase().includes('trash');
        }).length;
      });
      
      if (finalTrash < trashCount) {
        console.log('\n✅ SUCCESS! User create and delete both work!');
      } else {
        console.log('\n⚠️ Delete may have failed - trash buttons unchanged');
      }
    } else if (afterUsers > beforeUsers) {
      console.log('\n✅ User created! But no delete button (only admin can be deleted by non-admin)');
    } else {
      console.log('\n❌ User creation failed');
      // Check for error messages
      const errors = await page.evaluate(() => {
        return [...document.querySelectorAll('[class*="error"], [class*="red"]')].map(e => e.textContent.trim());
      });
      console.log('   Errors:', errors);
    }

  } catch (error) {
    console.error('\n💥 ERROR:', error.message);
    await page.screenshot({ path: 'test-screenshots/umgmt-error.png' });
  } finally {
    console.log('\n⏳ Browser open for 10s...');
    await page.waitForTimeout(10000);
    await browser.close();
  }
}

testUserManagement().catch(console.error);
