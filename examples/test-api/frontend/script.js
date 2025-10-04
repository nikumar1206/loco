const getBtn = document.getElementById('get-btn');
const postBtn = document.getElementById('post-btn');
const output = document.getElementById('output');

const API_URL = '/echo';

getBtn.addEventListener('click', async () => {
    try {
        const response = await fetch(API_URL);
        const data = await response.json();
        output.textContent = JSON.stringify(data, null, 2);
    } catch (error) {
        output.textContent = `Error: ${error.message}`;
    }
});

postBtn.addEventListener('click', async () => {
    try {
        const response = await fetch(API_URL, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ message: 'Hello from the frontend!' })
        });
        const data = await response.json();
        output.textContent = JSON.stringify(data, null, 2);
    } catch (error) {
        output.textContent = `Error: ${error.message}`;
    }
});
