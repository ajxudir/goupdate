import { useState } from 'react'

function App() {
  const [count, setCount] = useState(0)

  return (
    <div style={{ padding: '2rem', fontFamily: 'system-ui' }}>
      <h1>React Dashboard</h1>
      <p>A minimal React app for goupdate demo.</p>
      <button onClick={() => setCount(c => c + 1)}>
        Count: {count}
      </button>
    </div>
  )
}

export default App
