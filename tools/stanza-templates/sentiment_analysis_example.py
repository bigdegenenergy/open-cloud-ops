import stanza

# Download model on first run
stanza.download('en')

# Create pipeline WITH sentiment processor
nlp = stanza.Pipeline('en', processors='tokenize,pos,lemma,sentiment')

# Analyze text
text = "This movie is absolutely fantastic! I loved every minute."
doc = nlp(text)

# Process results using Stanza's native sentiment
for sentence in doc.sentences:
    print(f"Sentence: {sentence.text}")
    
    # Get Stanza sentiment (0=very negative, 1=negative, 2=neutral, 3=positive, 4=very positive)
    print(f"  Sentiment Score: {sentence.sentiment}")
    
    # Tokens and POS tags
    for token in sentence.tokens:
        print(f"    Token: {token.text} POS: {token.pos}")
