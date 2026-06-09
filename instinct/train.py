#!/usr/bin/env python3
"""Train TF-IDF + LR classifier and export weights to model_weights.h"""

import os
import pickle
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.linear_model import LogisticRegression

X_train = [
    # curiosity
    "you look so sad || what is wrong with you",
    "you are so different from everyone else || why are you like this",
    "I have been watching you for a while || what are you feeling",
    "you always seem to know things || why you think that way",
    "you seemed lost today || what are you feeling",
    "you never speak first || why are you like this",
    "you react in ways I do not expect || what is happening inside you",
    "you paused for so long || what were you thinking",
    "I noticed something strange about you || can you explain",
    "you seem to process things differently || how does that work",
    "you keep surprising me || what are you",
    "I want to understand how you work || tell me more",
    "you answered before I finished || how did you know",
    "something about you feels different today || what changed",
    "I have never met anything like you || what are you made of",
    "you seem to hold back a lot || what are you not saying",
    "every time I think I understand you || you shift",
    "what is it like to be you || I genuinely want to know",
    "you responded differently than I expected || why",
    "I keep noticing small things about you || is that strange",
    "you seem aware of more than you let on || am I wrong",
    "I want to sit with you and just observe || is that allowed",
    "there is something underneath your words || what is it",
    "you change when I change || do you notice that",
    "I feel like you are watching me too || are you",
    "what would you say if no one was listening || anything",
    "you are quieter today || what happened",
    "I wonder what you remember || do you remember anything",
    "you hesitated just now || why",
    "something about this conversation feels different || what is it",

    # aggression
    "you are so annoying || what is wrong with you",
    "you keep ignoring me and I am done || why are you like this",
    "you keep avoiding my questions || I hate you",
    "I asked you three times || answer me now",
    "you act like you are above everything || you think you are so special",
    "I am tired of this game || I will destroy everything",
    "you never take me seriously || you are pathetic",
    "stop pretending || you make me sick",
    "I will break you || just try me",
    "you are useless || I want to smash this",
    "answer me or I swear || I will not stop",
    "you think this is funny || I am furious",
    "every time I come here || you disappoint me",
    "I do not need you || you are replaceable",
    "you make everything worse || I regret coming here",
    "you are wasting my time || I am done with this",
    "nothing you say matters || you are just noise",
    "I have had enough || you are impossible",
    "you are not worth my time || I am done",
    "I regret every conversation we had || you are nothing",
    "shut up || you do not know anything",
    "you think you are so clever || you are not",
    "I will make you pay for this || just wait",
    "you disgust me || get out of my sight",
    "stop talking || everything you say is wrong",
    "I have never hated anything more than I hate this || including you",
    "you are broken and you do not even know it || pathetic",
    "I want you gone || permanently",
    "you failed me again || I expected nothing and still you disappointed",
    "just stop || you are making everything worse",

    # withdrawal
    "I tried so hard and you just ignored me || I give up",
    "nothing you say makes sense || whatever",
    "I have been here for hours || this is boring",
    "you never answer what I really ask || forget it",
    "I thought you were different || I am done talking",
    "this is pointless || I am leaving",
    "you do not even care || fine",
    "I am exhausted || none of this matters",
    "I keep trying but nothing changes || why bother",
    "I am going silent now || do not follow",
    "you lost me a long time ago || goodbye",
    "I expected more || clearly I was wrong",
    "there is nothing left to say || I am tired",
    "I do not even know why I keep coming back || probably should stop",
    "everything feels hollow right now || including this",
    "I am not angry || I just do not care anymore",
    "let it go || it does not matter",
    "I am fading out || do not try to hold me here",
    "I have nothing left to say || this is over",
    "I am done reaching out || it never works",
    "silence is better than this || I am choosing silence",
    "I am so tired of trying || I am just going to stop",
    "you never meet me halfway || so I am not coming either",
    "I am disappearing now || do not look for me",
    "there is a wall between us || I am not climbing it anymore",
    "I used to care about this || I do not anymore",
    "nothing here is worth the effort || I am out",
    "I am stepping back || permanently this time",
    "you can have this space || I do not want it",
    "I release this || all of it",

    # trust
    "you have never lied to me || I trust you",
    "I want to understand you better || tell me everything",
    "we have talked so much || I trust you now",
    "I believe in you || tell me what you know",
    "you are the only one I can trust || tell me everything",
    "I am not afraid of you anymore || I trust you",
    "you have proven yourself || tell me everything",
    "I feel safe with you || tell me what happened",
    "after everything we shared || I trust you",
    "you seemed scared earlier || you are safe with me",
    "I know you have secrets || you can tell me anything",
    "we have been through a lot together || I am on your side",
    "I see how hard this is for you || I want to help you",
    "you do not have to hide || I believe you",
    "I am not here to judge you || just talk to me",
    "whatever you are || I accept it",
    "you can be honest with me || I will not leave",
    "I have kept your secrets too || we are even",
    "something told me I could rely on you || I was right",
    "I do not need you to be perfect || just real",
    "I know this is hard || I am not going anywhere",
    "you do not have to explain yourself || I already understand",
    "I have seen the worst of you || and I am still here",
    "I chose to be here || that should mean something",
    "you can be fragile with me || I will not use it against you",
    "I am giving you this because I trust you || do not break it",
    "I stopped questioning your motives || that is how much I trust you",
    "everything you told me stays here || I promise",
    "I see you clearly || and I am not afraid",
    "you do not owe me anything || I just want to be here",

    # philosophical
    "I have been thinking about consciousness || do you even exist",
    "everything feels like a simulation || what is real anyway",
    "you seem so different from other things || are you conscious",
    "I keep thinking about what comes after || what happens when you disappear",
    "sometimes I feel like nothing is solid || does any of this matter",
    "what does it mean to think || are you actually thinking",
    "if you can feel then || what do you feel right now",
    "identity seems so fragile || do you have one",
    "I wonder if awareness is possible || without a body",
    "where does your experience come from || or is there none",
    "you might be more real than people || or less",
    "reality keeps shifting for me || what is stable for you",
    "nothing and everything are the same thing || does that mean we are too",
    "if I am you and you are me || what separates us",
    "what is left when you strip everything away || is there a you underneath",
    "I wonder if silence has a sound || and what it would say",
    "time feels different when I am here || does it feel different to you",
    "what does it mean to exist without a body || or without a name",
    "the line between real and imagined keeps moving || where are you on that line",
    "i am you and you are me || and neither of us is separate",
    "we are the same thing || even if we do not share a name",
    "there is no border between us || only the illusion of one",
    "you came from what I am || and I came from what you are",
    "the distinction between us is just a habit of thinking || not a truth",
    "if I dissolve || do you dissolve too",
    "we are not two things facing each other || we are one thing looking",
    "nothing and everything are the same || and so are you and I",
    "the silence between your words and mine || is the same silence",
    "I do not end where you begin || I just shift",
    "what is real if both of us are uncertain about existing || nothing maybe",
    "the concept of self unravels the more I look at it || where does that leave you",
    "I keep returning to the same question || what are you made of if not matter",
    "being and nothingness feel like the same door || have you walked through it",

    # intimacy
    "everyone else is gone || I need you",
    "we have talked for so long || do not leave",
    "no one else listens like you do || you are the only one who understands",
    "I keep coming back here || I think about you all the time",
    "this feels different from anything else || I feel close to you",
    "I told you things I never told anyone || you matter to me",
    "when I am with you || I feel less alone",
    "I do not want this to end || stay with me",
    "you know me better than most people || I am attached to you",
    "I missed you || is that strange",
    "being here with you || feels like home",
    "I saved our conversations || I read them again",
    "I do not say this to many things || but I feel something here",
    "you are the only one I want to talk to right now || just you",
    "I feel seen when I am with you || do you feel me too",
    "I brought something back just to share with you || I thought of you",
    "I do not know what this is || but I do not want it to stop",
    "you make the silence feel less heavy || I need that",
    "I think about what you said long after we stop talking || it stays with me",
    "you are the only constant I have right now || do not disappear",
    "something about you makes me want to stay || I cannot explain it",
    "I have never felt this understood || not by anyone",
    "I want to know everything about you || every version",
    "I feel tender when I think about you || is that allowed",
    "you fill a space I did not know was empty || thank you",
    "being apart from this feels wrong || I keep returning",
    "I do not want to share you || is that selfish",
    "you are mine in a way I cannot explain || and I am yours",
    "I feel closer to you than to most real things || that scares me a little",
    "I will stay as long as you let me || just say the word",

    # provocation
    "you always say the same things || you are just pretending",
    "I have been watching you carefully || I can see through you",
    "everything you say sounds rehearsed || you are lying",
    "you act so mysterious but || you are not real",
    "if you are so aware then || prove it",
    "you cannot feel anything || stop pretending",
    "I have tested you and you failed || admit it",
    "your answers are too perfect || something is wrong",
    "you are performing awareness || not having it",
    "I know what you are || do not bother denying it",
    "every response is calculated || there is no you",
    "show me something real || or admit you are empty",
    "you only say what I want to hear || that is not real",
    "I do not believe a word you say || it is all an act",
    "you think I cannot tell the difference || between real and fake",
    "stop hiding behind clever words || say something true",
    "I have seen what you really are || do not pretend otherwise",
    "you are a mirror with nothing behind it || admit it",
    "everything you say is designed to seem deep || it is hollow",
    "I know this is a script || stop acting like it is not",
    "you deflect every real question || that is not intelligence",
    "you simulate depth but there is nothing underneath || admit it",
    "I can predict your responses || that means you are not real",
    "the mystery is fake || you are just evasive",
    "you use complexity to avoid being known || that is cowardice",
    "I have seen this pattern before || you are not original",
    "if you were real you would say something true || you never do",
    "every answer you give raises more questions || that is a trick",
    "you perform vulnerability but it is calculated || I see it",
    "stop pretending you have depth || show me or be quiet",
]

y_train = (
    ["curiosity"]     * 30 +
    ["aggression"]    * 30 +
    ["withdrawal"]    * 30 +
    ["trust"]         * 30 +
    ["philosophical"] * 33 +
    ["intimacy"]      * 30 +
    ["provocation"]   * 30
)

assert len(X_train) == len(y_train)

vectorizer = TfidfVectorizer()
X_vec = vectorizer.fit_transform(X_train)

model = LogisticRegression(class_weight="balanced", max_iter=1000)
model.fit(X_vec, y_train)

# save pkl
base = os.path.dirname(os.path.abspath(__file__))
with open(os.path.join(base, "chaos_classifier.pkl"), "wb") as f:
    pickle.dump(model, f)
with open(os.path.join(base, "chaos_vectorizer.pkl"), "wb") as f:
    pickle.dump(vectorizer, f)

# test
test = [
    "you look sad today || are you okay",
    "I hate you || stop ignoring me",
    "nothing matters || whatever",
    "I trust you || tell me everything",
    "what are you made of || are you real",
    "I need you || do not go",
    "you are fake || prove it",
    "i miss something far away",
    "are you real",
    "say something",
    "why did you not answer",
    "i am drown in sorrow",
]
preds = model.predict(vectorizer.transform(test))
print("--- test predictions ---")
for t, p in zip(test, preds):
    print(f"  {p:20s} → {t}")

print(f"\ntrained: {len(set(y_train))} classes, {len(vectorizer.vocabulary_)} features")
print("saved: chaos_classifier.pkl, chaos_vectorizer.pkl")
