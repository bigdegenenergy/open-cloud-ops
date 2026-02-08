import edu.stanford.nlp.pipeline.*;
import edu.stanford.nlp.ling.*;
import java.util.*;

public class SentimentAnalysisExample {
    public static void main(String[] args) {
        // Create pipeline with sentiment annotator
        Properties props = new Properties();
        props.setProperty("annotators", "tokenize,ssplit,pos,lemma,sentiment");
        StanfordCoreNLP pipeline = new StanfordCoreNLP(props);
        
        // Analyze text
        String text = "This movie is absolutely fantastic! I loved every minute.";
        CoreDocument document = new CoreDocument(text);
        pipeline.annotate(document);
        
        // Process results
        for (CoreSentence sentence : document.sentences()) {
            System.out.println("Sentence: " + sentence.text());
            
            // Get sentiment as numeric value (0-4 scale)
            // Note: sentimentValue() returns int; sentiment() returns string label
            try {
                // Safe null checks before accessing sentiment values
                Integer sentimentScore = sentence.sentimentValue();
                String sentimentLabel = sentence.sentiment();
                
                if (sentimentScore != null && sentimentLabel != null) {
                    System.out.println("  Sentiment: " + sentimentScore + " (" + sentimentLabel + ")");
                } else {
                    System.out.println("  Sentiment: Not available (values are null)");
                }
            } catch (Exception e) {
                System.out.println("  Sentiment: Not available (exception: " + e.getMessage() + ")");
            }
            
            // Tokens and POS tags
            for (CoreLabel token : sentence.tokens()) {
                System.out.println("    Token: " + token.word() + 
                                 " POS: " + token.pos());
            }
        }
    }
}
