import { useState, useEffect } from 'react'

function App() {
  const [query, setQuery] = useState('')
  const [suggestions, setSuggestions] = useState([])
  const [dummyResponse, setDummyResponse] = useState('')
  const [trending, setTrending] = useState([])

  // Fetch Trending Searches (Now expects objects with {query, count})
  useEffect(() => {
    const fetchTrending = () => {
      fetch('http://localhost:8080/trending')
        .then(res => res.json())
        .then(data => setTrending(data || []))
        .catch(err => console.error(err))
    }
    fetchTrending()
    const interval = setInterval(fetchTrending, 3000)
    return () => clearInterval(interval)
  }, [])

  // Debounced Suggestion Fetch
  useEffect(() => {
    if (!query) return

    const timer = setTimeout(() => {
      fetch(`http://localhost:8080/suggest?q=${query}`)
        .then(res => res.json())
        .then(data => setSuggestions(data || []))
        .catch(err => console.error(err))
    }, 300)

    return () => clearTimeout(timer)
  }, [query])

  const handleInputChange = (e) => {
    const val = e.target.value
    setQuery(val)
    if (!val) setSuggestions([])
  }

  const handleSearch = (searchQuery) => {
    if (!searchQuery) return

    fetch('http://localhost:8080/search', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query: searchQuery.toLowerCase() })
    })
      .then(res => res.json())
      .then(() => {
        setDummyResponse(`Searched for "${searchQuery}" successfully.`)
        setSuggestions([])
        setQuery('') 
        
        // Clear success message after 3 seconds
        setTimeout(() => setDummyResponse(''), 3000)
      })
      .catch(err => console.error(err))
  }

  // Format large numbers (e.g., 150000 -> 150k)
  const formatCount = (num) => {
    return num >= 1000 ? (num / 1000).toFixed(1) + 'k' : num
  }

  return (
    <div style={{ 
      fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif', 
      maxWidth: '600px', 
      margin: '60px auto',
      padding: '0 20px'
    }}>
      <h2 style={{ textAlign: 'center', color: '#1a1a1a', marginBottom: '30px' }}>
        Search Explorer
      </h2>
      
      {/* Search Input Area */}
      <div style={{ position: 'relative' }}>
        <div style={{ display: 'flex', gap: '10px' }}>
          <input
            type="text"
            value={query}
            onChange={handleInputChange}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch(query)}
            placeholder="Type to search..."
            style={{ 
              flex: 1, 
              padding: '14px 20px', 
              fontSize: '16px', 
              borderRadius: '8px', 
              border: '1px solid #ddd',
              outline: 'none',
              boxShadow: '0 2px 4px rgba(0,0,0,0.02)'
            }}
          />
          <button 
            onClick={() => handleSearch(query)}
            style={{ 
              padding: '0 24px', 
              fontSize: '16px', 
              fontWeight: '600', 
              color: 'white', 
              background: '#0066ff', 
              border: 'none', 
              borderRadius: '8px', 
              cursor: 'pointer',
              transition: 'background 0.2s'
            }}
            onMouseEnter={(e) => e.target.style.background = '#0052cc'}
            onMouseLeave={(e) => e.target.style.background = '#0066ff'}
          >
            Search
          </button>
        </div>
        
        {/* Suggestion Dropdown */}
        {suggestions.length > 0 && (
          <ul style={{
            position: 'absolute', top: '100%', left: 0, right: '110px', 
            background: 'white', listStyle: 'none', margin: '8px 0 0 0', padding: 0,
            border: '1px solid #eee', borderRadius: '8px',
            boxShadow: '0 10px 15px -3px rgba(0,0,0,0.1)', zIndex: 10,
            overflow: 'hidden'
          }}>
            {suggestions.map((item, index) => (
              <li 
                key={index}
                onClick={() => handleSearch(item.query)}
                style={{ 
                  padding: '12px 20px', cursor: 'pointer', 
                  borderBottom: index === suggestions.length - 1 ? 'none' : '1px solid #f5f5f5',
                  color: '#333', fontSize: '15px'
                }}
                onMouseEnter={(e) => e.target.style.background = '#f8f9fa'}
                onMouseLeave={(e) => e.target.style.background = 'white'}
              >
                {item.query}
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* Success Message Toast */}
      {dummyResponse && (
        <div style={{ 
          marginTop: '15px', padding: '12px 20px', background: '#ecfdf5', 
          color: '#065f46', borderRadius: '6px', fontSize: '14px', border: '1px solid #a7f3d0'
        }}>
          ✅ {dummyResponse}
        </div>
      )}

      {/* Trending Searches Section */}
      <div style={{ 
        marginTop: '40px', padding: '24px', background: '#f8f9fa', 
        borderRadius: '12px', border: '1px solid #eaeaea'
      }}>
        <h3 style={{ margin: '0 0 16px 0', fontSize: '16px', color: '#444', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span>🔥</span> Trending Searches
        </h3>
        
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '10px' }}>
          {trending.map((trend, idx) => (
            <button 
              key={idx} 
              onClick={() => handleSearch(trend.query)}
              style={{ 
                display: 'flex', alignItems: 'center', gap: '8px',
                background: 'white', padding: '8px 16px', borderRadius: '20px', 
                fontSize: '14px', cursor: 'pointer', border: '1px solid #ddd',
                color: '#333', transition: 'all 0.2s ease',
                boxShadow: '0 1px 2px rgba(0,0,0,0.05)'
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.borderColor = '#0066ff';
                e.currentTarget.style.transform = 'translateY(-1px)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.borderColor = '#ddd';
                e.currentTarget.style.transform = 'translateY(0)';
              }}
            >
              <span style={{ fontWeight: '500' }}>{trend.query}</span>
              <span style={{ 
                background: '#f1f3f5', color: '#666', padding: '2px 8px', 
                borderRadius: '10px', fontSize: '12px', fontWeight: '600' 
              }}>
                {formatCount(trend.count)}
              </span>
            </button>
          ))}
        </div>
      </div>

    </div>
  )
}

export default App